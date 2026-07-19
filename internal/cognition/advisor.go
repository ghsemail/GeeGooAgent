package cognition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AdvisorConfig configures the optional HTTP advisor sidecar.
type AdvisorConfig struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// AdvisorClient calls the Python advisor sidecar (suggestion-only).
type AdvisorClient struct {
	baseURL string
	http    *http.Client
}

// NewAdvisorClient creates a client. baseURL may be empty (calls will fail fast).
func NewAdvisorClient(cfg AdvisorConfig) *AdvisorClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: timeout}
	}
	return &AdvisorClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		http:    hc,
	}
}

type rankRequest struct {
	Items []RankItem `json:"items"`
}

type rankResponse struct {
	Items  []RankItem     `json:"items"`
	Reason string         `json:"reason,omitempty"`
	Extra  map[string]any `json:"-"`
}

type evalRequest struct {
	SessionID     string   `json:"session_id"`
	AssistantText string   `json:"assistant_text"`
	Failed        bool     `json:"failed"`
	ToolNames     []string `json:"tool_names,omitempty"`
}

type evalResponse struct {
	Accept         bool   `json:"accept"`
	RetrySuggested bool   `json:"retry_suggested"`
	Reason         string `json:"reason,omitempty"`
}

var forbiddenAdvisorKeys = []string{
	"tool_calls", "tool_call", "state", "workflow", "workflow_decision",
	"mutate", "session_write", "pending_plan",
}

// Rank calls POST /v1/advisor/rank.
func (c *AdvisorClient) Rank(ctx context.Context, items []RankItem) ([]RankItem, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("advisor: not configured")
	}
	var resp rankResponse
	if err := c.post(ctx, "/v1/advisor/rank", rankRequest{Items: items}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return items, nil
	}
	return resp.Items, nil
}

// Evaluate calls POST /v1/advisor/evaluate.
func (c *AdvisorClient) Evaluate(ctx context.Context, in EvalInput) (EvalResult, error) {
	if c == nil || c.baseURL == "" {
		return EvalResult{}, fmt.Errorf("advisor: not configured")
	}
	var resp evalResponse
	req := evalRequest{
		SessionID: in.SessionID, AssistantText: in.AssistantText,
		Failed: in.Failed, ToolNames: in.ToolNames,
	}
	if err := c.post(ctx, "/v1/advisor/evaluate", req, &resp); err != nil {
		return EvalResult{}, err
	}
	return EvalResult{
		Accept: resp.Accept, RetrySuggested: resp.RetrySuggested, Reason: resp.Reason,
	}, nil
}

func (c *AdvisorClient) post(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("advisor: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("advisor: decode: %w", err)
	}
	return rejectForbiddenJSON(b)
}

func rejectForbiddenJSON(raw []byte) error {
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil
	}
	return rejectForbiddenPayload(probe)
}

func rejectForbiddenPayload(v any) error {
	switch t := v.(type) {
	case map[string]any:
		for k := range t {
			kl := strings.ToLower(k)
			for _, bad := range forbiddenAdvisorKeys {
				if kl == bad {
					return fmt.Errorf("advisor: forbidden response field %q", k)
				}
			}
		}
		for _, child := range t {
			if err := rejectForbiddenPayload(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range t {
			if err := rejectForbiddenPayload(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// AdvisorRanker delegates to the sidecar; falls back to IdentityRanker on error.
type AdvisorRanker struct {
	Client   *AdvisorClient
	Fallback Ranker
}

// Rank implements Ranker.
func (r AdvisorRanker) Rank(ctx context.Context, items []RankItem) ([]RankItem, error) {
	fb := r.Fallback
	if fb == nil {
		fb = IdentityRanker{}
	}
	if r.Client == nil {
		return fb.Rank(ctx, items)
	}
	out, err := r.Client.Rank(ctx, items)
	if err != nil {
		return fb.Rank(ctx, items)
	}
	return out, nil
}

// AdvisorEvaluator delegates to the sidecar; falls back to AcceptAllEvaluator on error.
type AdvisorEvaluator struct {
	Client   *AdvisorClient
	Fallback Evaluator
}

// Evaluate implements Evaluator.
func (e AdvisorEvaluator) Evaluate(ctx context.Context, in EvalInput) (EvalResult, error) {
	fb := e.Fallback
	if fb == nil {
		fb = AcceptAllEvaluator{}
	}
	if e.Client == nil {
		return fb.Evaluate(ctx, in)
	}
	res, err := e.Client.Evaluate(ctx, in)
	if err != nil {
		return fb.Evaluate(ctx, in)
	}
	return res, nil
}

// BundleWithAdvisor overlays optional advisor Ranker/Evaluator on defaults.
func BundleWithAdvisor(client *AdvisorClient, useRanker, useEvaluator bool) Bundle {
	b := Defaults()
	if client == nil {
		return b
	}
	if useRanker {
		b.Ranker = AdvisorRanker{Client: client, Fallback: IdentityRanker{}}
	}
	if useEvaluator {
		b.Evaluator = AdvisorEvaluator{Client: client, Fallback: AcceptAllEvaluator{}}
	}
	return b
}
