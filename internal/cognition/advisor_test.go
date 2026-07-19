package cognition_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func TestAdvisorRankerFallsBackOnError(t *testing.T) {
	t.Parallel()
	r := cognition.AdvisorRanker{Client: cognition.NewAdvisorClient(cognition.AdvisorConfig{BaseURL: "http://127.0.0.1:1"})}
	items := []cognition.RankItem{{ID: "a"}, {ID: "b"}}
	out, err := r.Rank(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].ID != "a" {
		t.Fatalf("fallback=%+v", out)
	}
}

func TestAdvisorClientRankAndEvaluate(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/advisor/rank":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{"id": "b", "text": "second", "score": 0.9}},
			})
		case "/v1/advisor/evaluate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accept": false, "retry_suggested": true, "reason": "too short",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := cognition.NewAdvisorClient(cognition.AdvisorConfig{BaseURL: srv.URL})
	out, err := client.Rank(context.Background(), []cognition.RankItem{
		{ID: "a", Score: 0.1}, {ID: "b", Score: 0.2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "b" {
		t.Fatalf("rank=%+v", out)
	}

	res, err := client.Evaluate(context.Background(), cognition.EvalInput{AssistantText: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accept || !res.RetrySuggested {
		t.Fatalf("eval=%+v", res)
	}
}

func TestAdvisorRejectsForbiddenFields(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accept": true, "tool_calls": []any{map[string]any{"name": "evil"}},
		})
	}))
	defer srv.Close()
	client := cognition.NewAdvisorClient(cognition.AdvisorConfig{BaseURL: srv.URL})
	_, err := client.Evaluate(context.Background(), cognition.EvalInput{AssistantText: "x"})
	if err == nil {
		t.Fatal("expected forbidden field error")
	}
}

func TestBundleWithAdvisorPartial(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"accept": true})
	}))
	defer srv.Close()
	client := cognition.NewAdvisorClient(cognition.AdvisorConfig{BaseURL: srv.URL})
	b := cognition.BundleWithAdvisor(client, false, true)
	if _, ok := b.Evaluator.(cognition.AdvisorEvaluator); !ok {
		t.Fatalf("evaluator type %T", b.Evaluator)
	}
	if _, ok := b.Ranker.(cognition.IdentityRanker); !ok {
		t.Fatalf("ranker type %T", b.Ranker)
	}
}
