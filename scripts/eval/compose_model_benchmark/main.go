// Benchmark signal_diagnose compose models from ops catalog (real prompt, stream timing).
//
// Usage:
//
//	GEEGOO_CONFIG=/path/to/config.json go run ./scripts/eval/compose_model_benchmark/
//
// Writes JSON lines: model, first_visible_ms, total_ms, chars, error.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

type row struct {
	Model          string `json:"model"`
	Display        string `json:"display_name,omitempty"`
	ForcedReason   bool   `json:"forced_reasoning_catalog"`
	FirstVisibleMS int64  `json:"first_visible_ms"`
	TotalMS        int64  `json:"total_ms"`
	Chars          int    `json:"chars"`
	Error          string `json:"error,omitempty"`
}

func main() {
	path := strings.TrimSpace(os.Getenv("GEEGOO_CONFIG"))
	if path == "" {
		fmt.Fprintln(os.Stderr, "GEEGOO_CONFIG required")
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(path, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	candidates, err := application.ListCatalogComposeCandidates(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list catalog:", err)
		os.Exit(1)
	}
	msgs := workflowchat.BuildSignalDiagnoseComposeMessages(workflowchat.BenchmarkComposeFlow())
	fmt.Fprintf(os.Stderr, "catalog compose candidates: %d\n", len(candidates))
	var out []row
	for _, doc := range candidates {
		r := benchmarkOne(ctx, application, doc, msgs)
		out = append(out, r)
		b, _ := json.Marshal(r)
		fmt.Println(string(b))
	}
	_ = os.WriteFile("/opt/cursor/artifacts/compose_model_benchmark.json", mustJSON(out), 0o644)
}

func benchmarkOne(ctx context.Context, a *app.App, doc admin.ConfiguredModel, msgs []llm.Message) row {
	r := row{
		Model:   strings.TrimSpace(doc.Name),
		Display: strings.TrimSpace(doc.DisplayName),
		ForcedReason: llm.ModelCatalogLikelyForcedReasoning(doc.Name+" "+doc.DisplayName),
	}
	p, err := a.ComposeBenchmarkProvider(doc)
	if err != nil || p == nil {
		r.Error = fmt.Sprint(err)
		return r
	}
	streamer, ok := p.(llm.Streamer)
	if !ok {
		r.Error = "not a streamer"
		return r
	}
	callCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	callCtx = llm.WithCallMeta(callCtx, llm.CallMeta{
		Kind: llm.TaskClassify, ThinkingOff: true, MaxTokensOverride: 4096,
	})
	start := time.Now()
	var firstVisible time.Time
	var acc llm.AnswerStreamAccumulator
	acc.OnVisible = func(string) {
		if firstVisible.IsZero() {
			firstVisible = time.Now()
		}
	}
	_, err = streamer.ChatStream(callCtx, msgs, nil, 0, 4096, func(delta llm.StreamDelta) {
		if strings.TrimSpace(delta.Content) != "" {
			acc.PushContent(delta.Content)
		}
	})
	r.TotalMS = time.Since(start).Milliseconds()
	if !firstVisible.IsZero() {
		r.FirstVisibleMS = firstVisible.Sub(start).Milliseconds()
	}
	text := llm.NormalizeComposeMarkdown(acc.Visible())
	r.Chars = len([]rune(text))
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

func mustJSON(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
