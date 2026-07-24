package runtimeapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func (h *Handler) registerCompareRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/dashboard/compare/history", h.compareHistory)
	mux.HandleFunc("POST /v1/dashboard/compare/stream", h.compareStream)
	mux.HandleFunc("POST /v1/dashboard/compare/clear", h.compareClear)
}

type compareRunResult struct {
	Spec       string          `json:"spec"`
	Provider   string          `json:"provider"`
	Model      string          `json:"model"`
	Reply      string          `json:"reply,omitempty"`
	LatencyMS  int             `json:"latency_ms,omitempty"`
	TokensIn   int             `json:"tokens_in,omitempty"`
	TokensOut  int             `json:"tokens_out,omitempty"`
	CostUSD    float64         `json:"cost_usd,omitempty"`
	Iterations int             `json:"iterations,omitempty"`
	Error      string          `json:"error,omitempty"`
	Gate       string          `json:"gate,omitempty"`
	Tools      []string        `json:"tools,omitempty"`
	Quality    *compareQuality `json:"quality,omitempty"`
}

type compareQuality struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
	Judge  string  `json:"judge,omitempty"`
}

type compareRun struct {
	TS      string             `json:"ts"`
	Message string             `json:"message"`
	Results []compareRunResult `json:"results"`
}

func (h *Handler) compareStorePath() string {
	if h.App != nil && h.App.Workspace != "" {
		return filepath.Join(h.App.Workspace, "compare", "history.jsonl")
	}
	return filepath.Join(os.TempDir(), "geegoo_compare_history.jsonl")
}

func (h *Handler) loadCompareRuns() []compareRun {
	path := h.compareStorePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	runs := []compareRun{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var run compareRun
		if json.Unmarshal([]byte(line), &run) == nil {
			runs = append(runs, run)
		}
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].TS > runs[j].TS })
	return runs
}

func (h *Handler) appendCompareRun(run compareRun) error {
	path := h.compareStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	return err
}

func (h *Handler) compareHistory(w http.ResponseWriter, r *http.Request) {
	runs := h.loadCompareRuns()
	writeJSON(w, map[string]any{
		"runs":      runs,
		"aggregate": aggregateCompareRuns(runs),
	})
}

func (h *Handler) compareClear(w http.ResponseWriter, r *http.Request) {
	path := h.compareStorePath()
	_ = os.Remove(path)
	writeJSON(w, map[string]any{"runs": []compareRun{}, "aggregate": []map[string]any{}})
}

type compareStreamRequest struct {
	Message    string   `json:"message"`
	Models     []string `json:"models"`
	Judge      bool     `json:"judge"`
	JudgeModel string   `json:"judge_model"`
}

func (h *Handler) compareStream(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
	var req compareStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" || len(req.Models) == 0 {
		writeError(w, http.StatusBadRequest, "message and models required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	run := compareRun{TS: time.Now().UTC().Format(time.RFC3339Nano), Message: message}
	for _, spec := range req.Models {
		provider, model := splitModelSpec(spec)
		writeCompareSSE(w, flusher, map[string]any{"kind": "start", "spec": spec, "provider": provider, "model": model})
		start := time.Now()
		resp, err := h.App.Gateway.Chat(r.Context(), []llm.Message{
			{Role: llm.RoleUser, Content: message},
		}, nil, "compare-"+spec, 0)
		latency := int(time.Since(start).Milliseconds())
		result := compareRunResult{Spec: spec, Provider: provider, Model: model, LatencyMS: latency}
		if err != nil {
			result.Error = err.Error()
		} else if resp != nil {
			result.Reply = resp.Content
			result.TokensIn = resp.Usage.PromptTokens
			result.TokensOut = resp.Usage.CompletionTokens
			result.Iterations = 1
		}
		run.Results = append(run.Results, result)
		writeCompareSSE(w, flusher, map[string]any{
			"kind": "result", "spec": spec, "provider": provider, "model": model,
			"reply": result.Reply, "latency_ms": result.LatencyMS,
			"tokens_in": result.TokensIn, "tokens_out": result.TokensOut, "error": result.Error,
		})
	}
	_ = h.appendCompareRun(run)
	writeCompareSSE(w, flusher, map[string]any{"kind": "done"})
}

func splitModelSpec(spec string) (string, string) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "geegoo", spec
}

func writeCompareSSE(w http.ResponseWriter, flusher http.Flusher, payload map[string]any) {
	raw, _ := json.Marshal(payload)
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(raw)
	_, _ = w.Write([]byte("\n\n"))
	flusher.Flush()
}

func aggregateCompareRuns(runs []compareRun) []map[string]any {
	type agg struct {
		spec, provider, model string
		runs, ok              int
		totalLatency          int
		totalIn, totalOut     int
		totalCost             float64
	}
	m := map[string]*agg{}
	for _, run := range runs {
		for _, res := range run.Results {
			a := m[res.Spec]
			if a == nil {
				a = &agg{spec: res.Spec, provider: res.Provider, model: res.Model}
				m[res.Spec] = a
			}
			a.runs++
			if res.Error == "" {
				a.ok++
				a.totalLatency += res.LatencyMS
				a.totalIn += res.TokensIn
				a.totalOut += res.TokensOut
				a.totalCost += res.CostUSD
			}
		}
	}
	out := []map[string]any{}
	for _, a := range m {
		out = append(out, map[string]any{
			"spec": a.spec, "provider": a.provider, "model": a.model,
			"runs": a.runs, "ok": a.ok, "total_latency_ms": a.totalLatency,
			"total_tokens_in": a.totalIn, "total_tokens_out": a.totalOut,
			"total_tokens": a.totalIn + a.totalOut, "total_cost_usd": a.totalCost,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["total_cost_usd"].(float64) < out[j]["total_cost_usd"].(float64)
	})
	return out
}
