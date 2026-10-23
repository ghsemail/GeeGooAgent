package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildPriceAnalysisJEVStateIncludesRows(t *testing.T) {
	pa := &PriceAnalysis{
		Code: "00700.HK",
		Rows: []PriceAnalysisRow{
			{Timeframe: "日线", Conclusion: "偏多", Structure: "HH/HL", BosChoch: "BOS↑", Position: "中上", ER: 0.42},
		},
		PriceScore: 12.5,
		ScoreReason: "周线日线同向",
	}
	state := BuildPriceAnalysisJEVState(pa, "buy")
	if !strings.Contains(state, "00700.HK") || !strings.Contains(state, "日线") || !strings.Contains(state, "BOS↑") {
		t.Fatalf("state missing PA fields: %q", state)
	}
	if !strings.Contains(state, "买入触发") {
		t.Fatal("expected buy side in state")
	}
}

func mockDecisionChoiceResponse(w http.ResponseWriter, answers map[string]any) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    0,
		"message": "ok",
		"data": map[string]any{
			"model":   "jev-1.13.0",
			"answers": answers,
		},
	})
}

func TestResolvePriceAnalysisJEVSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/decision/choice" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["state"] == nil {
			t.Fatal("expected state in request")
		}
		mockDecisionChoiceResponse(w, map[string]any{
			"direction": map[string]any{
				"type":   "choice",
				"choice": "看多",
				"probabilities": map[string]float64{
					"看多": 0.62, "看空": 0.18, "不明确": 0.20,
				},
				"confidence": 0.71,
			},
		})
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{
		Code: "00700.HK",
		Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多", Structure: "HL", BosChoch: "-", Position: "中"}},
	}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "buy")
	if !out.OK {
		t.Fatalf("ok=false err=%q", out.Error)
	}
	if out.Source != "jev" || out.RecommendedChoice != "看多" || out.RecommendedIndex != 0 {
		t.Fatalf("unexpected out: %+v", out)
	}
	if len(out.Choices) != 3 || out.Choices[0].Probability != 0.62 {
		t.Fatalf("choices=%+v", out.Choices)
	}
	if out.RecommendedReason == "" {
		t.Fatal("expected reason")
	}
}

func TestResolvePriceAnalysisJEVFailureNoFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Code: "X", Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多"}}}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "sell")
	if out.OK {
		t.Fatal("expected ok=false on http error")
	}
	if out.Error == "" {
		t.Fatal("expected error message")
	}
	if out.RecommendedChoice != "" {
		t.Fatalf("must not fake choice on failure, got %q", out.RecommendedChoice)
	}
}

func TestResolvePriceAnalysisJEVNotConfigured(t *testing.T) {
	pa := &PriceAnalysis{Code: "X", Rows: []PriceAnalysisRow{{Timeframe: "日线"}}}
	out := ResolvePriceAnalysisJEV(context.Background(), NewJevClient(JevConfig{}), pa, "buy")
	if out.OK || !strings.Contains(out.Error, "jev not configured") {
		t.Fatalf("got %+v", out)
	}
}

func TestApplyPriceAnalysisJEVDualWrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mockDecisionChoiceResponse(w, map[string]any{
			"direction": map[string]any{
				"type": "choice", "choice": "不明确",
				"probabilities": map[string]float64{"看多": 0.2, "看空": 0.2, "不明确": 0.6},
				"confidence":    0.5,
			},
		})
	}))
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	detail := LabEpisodeDetail{PriceAnalysis: &PriceAnalysis{Code: "Y", Rows: []PriceAnalysisRow{{Timeframe: "5m"}}}}
	applyPriceAnalysisJEV(context.Background(), &detail, "buy", jev)
	if detail.PriceAnalysisJEV == nil || detail.PriceAnalysis.PriceAnalysisJEV == nil {
		t.Fatal("expected dual write")
	}
	if detail.PriceAnalysisJEV != detail.PriceAnalysis.PriceAnalysisJEV {
		t.Fatal("expected same pointer dual write")
	}
	if !detail.PriceAnalysisJEV.OK || detail.PriceAnalysisJEV.RecommendedChoice != "不明确" {
		t.Fatalf("jev=%+v", detail.PriceAnalysisJEV)
	}
}
