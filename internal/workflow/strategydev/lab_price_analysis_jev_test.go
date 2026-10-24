package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestBuildPriceAnalysisJEVStateIncludesRows(t *testing.T) {
	pa := &PriceAnalysis{
		Code: "00700.HK",
		Rows: []PriceAnalysisRow{
			{Timeframe: "日线", Conclusion: "偏多", Structure: "HH/HL", BosChoch: "BOS↑", Position: "中上", ER: 0.42},
		},
		PriceScore:  12.5,
		ScoreReason: "周线日线同向",
	}
	state := BuildPriceAnalysisJEVState(pa, "buy", nil)
	if !strings.Contains(state, "00700.HK") || !strings.Contains(state, "日线") || !strings.Contains(state, "BOS↑") {
		t.Fatalf("state missing PA fields: %q", state)
	}
	if !strings.Contains(state, "买入触发") {
		t.Fatal("expected buy side in state")
	}
}

func TestBuildPriceAnalysisJEVStateIncludesKlinePatterns(t *testing.T) {
	pa := &PriceAnalysis{Code: "X", Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多"}}}
	kline := &LabJEVKlineInput{
		Summary: "锤子(看多)",
		Bias:    &slots.KlinePatternBias{Bias: "看多", Score: 1},
		Hits:    []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
	}
	state := BuildPriceAnalysisJEVState(pa, "buy", kline)
	if !strings.Contains(state, "锤子") || !strings.Contains(state, "形态摘要") {
		t.Fatalf("state missing kline: %q", state)
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

func jevTestServer(t *testing.T, choice string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"choice": choice,
				"probabilities": map[string]float64{
					"看多": 0.62, "看空": 0.18, "不明确": 0.20,
				},
				"confidence": 0.71,
			},
		})
	}))
}

func TestResolvePriceAnalysisJEVSuccess(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{
		Code: "00700.HK",
		Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多", Structure: "HL", BosChoch: "-", Position: "中"}},
	}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "buy", nil)
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

func TestResolvePriceAnalysisJEVBothLegsSameDirection(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多"}}}
	kline := &LabJEVKlineInput{
		Summary: "吞没(看多)",
		Bias:    &slots.KlinePatternBias{Bias: "看多"},
		Hits:    []slots.KlinePatternHit{{LabelCN: "吞没", Direction: 1}},
	}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "buy", kline)
	if !out.OK || out.RecommendedChoice != "看多" {
		t.Fatalf("got %+v", out)
	}
}

func TestResolvePriceAnalysisJEVConflictPicksSideNotUnclear(t *testing.T) {
	srv := jevTestServer(t, "看空")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多"}}}
	kline := &LabJEVKlineInput{
		Bias: &slots.KlinePatternBias{Bias: "看空"},
		Hits: []slots.KlinePatternHit{{LabelCN: "射击之星", Direction: -1}},
	}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "buy", kline)
	if !out.OK || out.RecommendedChoice != "看空" {
		t.Fatalf("conflict mock should pick 看空, got %+v", out)
	}
}

func TestResolvePriceAnalysisJEVPatternOnly(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	kline := &LabJEVKlineInput{
		Summary: "锤子",
		Bias:    &slots.KlinePatternBias{Bias: "看多"},
		Hits:    []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
	}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, nil, "sell", kline)
	if !out.OK || out.RecommendedChoice != "看多" {
		t.Fatalf("pattern-only: %+v", out)
	}
}

func TestResolvePriceAnalysisJEVPAOnly(t *testing.T) {
	srv := jevTestServer(t, "看空")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Rows: []PriceAnalysisRow{{Timeframe: "周线", Conclusion: "偏空"}}}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "buy", nil)
	if !out.OK || out.RecommendedChoice != "看空" {
		t.Fatalf("pa-only: %+v", out)
	}
}

func TestResolvePriceAnalysisJEVInsufficientState(t *testing.T) {
	jev := NewJevClient(JevConfig{BaseURL: "http://unused", APIKey: "k", Model: "jev-1.13.0"})
	out := ResolvePriceAnalysisJEV(context.Background(), jev, nil, "buy", nil)
	if out.OK || !strings.Contains(out.Error, "empty") {
		t.Fatalf("got %+v", out)
	}
	paEmpty := &PriceAnalysis{Code: "only-code"}
	out2 := ResolvePriceAnalysisJEV(context.Background(), jev, paEmpty, "buy", nil)
	if out2.OK || !strings.Contains(out2.Error, "empty") {
		t.Fatalf("non-informative PA: %+v", out2)
	}
}

func TestResolvePriceAnalysisJEVFailureNoFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Code: "X", Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多"}}}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "sell", nil)
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
	out := ResolvePriceAnalysisJEV(context.Background(), NewJevClient(JevConfig{}), pa, "buy", nil)
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
	detail := LabEpisodeDetail{PriceAnalysis: &PriceAnalysis{Code: "Y", Rows: []PriceAnalysisRow{{Timeframe: "5m", Conclusion: "震荡"}}}}
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

func TestApplyPriceAnalysisJEVPatternOnlyOnDetail(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	detail := LabEpisodeDetail{
		KlinePatternSummary: "锤子(看多)",
		KlinePatternBias:    &slots.KlinePatternBias{Bias: "看多"},
		KlinePatterns:       []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
	}
	applyPriceAnalysisJEV(context.Background(), &detail, "buy", jev)
	if detail.PriceAnalysisJEV == nil || !detail.PriceAnalysisJEV.OK {
		t.Fatalf("pattern-only detail JEV: %+v", detail.PriceAnalysisJEV)
	}
}
