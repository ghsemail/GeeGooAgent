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
	if pa.PriceScore != 12.5 {
		t.Fatal("PA object must retain price_score for UI JSON")
	}
	for _, forbidden := range []string{"综合分", "分项[", "price_score", "tf_scores", "子分="} {
		if strings.Contains(state, forbidden) {
			t.Fatalf("JEV state must omit composite scores, found %q in:\n%s", forbidden, state)
		}
	}
}

func TestBuildPriceAnalysisJEVStateOmitsTFScoreBreakdown(t *testing.T) {
	pa := &PriceAnalysis{
		Code:       "X",
		PriceScore: 42.5,
		ScoreReason: "周线日线同向",
		Rows:        []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多", Structure: "HL", BosChoch: "-", Position: "中"}},
		TFScores: map[string]TFScoreBreakdown{
			"周线": {Subscore: 0.8, Conclusion: 0.5, Position: 0.3, BosChoch: 0.2},
		},
	}
	state := BuildPriceAnalysisJEVState(pa, "sell", nil)
	if !strings.Contains(state, "[60m]") {
		t.Fatalf("expected structure row: %q", state)
	}
	for _, forbidden := range []string{"综合分", "42.50", "42.5", "分项[", "子分="} {
		if strings.Contains(state, forbidden) {
			t.Fatalf("forbidden %q in state:\n%s", forbidden, state)
		}
	}
	if len(pa.TFScores) != 1 || pa.PriceScore != 42.5 {
		t.Fatal("PA JSON fields must remain on struct")
	}
}

func TestBuildPriceAnalysisJEVStateIncludesKlinePatterns(t *testing.T) {
	pa := &PriceAnalysis{Code: "X", Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多"}}}
	kline := &LabJEVKlineInput{
		Summary: "锤子(看多)",
		Bias:    &slots.KlinePatternBias{Bias: "看多", Score: 1},
		Hits:    []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
		SignalFreq: "5m",
	}
	state := BuildPriceAnalysisJEVState(pa, "buy", kline)
	if !strings.Contains(state, "锤子") || !strings.Contains(state, "形态摘要") {
		t.Fatalf("state missing kline: %q", state)
	}
	if !strings.Contains(state, "5m 锚点命中") {
		t.Fatalf("expected signal anchor tag: %q", state)
	}
}

func TestBuildPriceAnalysisJEVStateIncludesMTFTaggedHits(t *testing.T) {
	pa := &PriceAnalysis{Code: "00700.HK", Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多"}}}
	detail := &LabEpisodeDetail{
		KlinePatternMTF: &slots.KlinePatternMTFContext{
			SignalFrequency: "5m",
			WeightSchema:    "mtf_v1",
			Timeframes: []slots.KlinePatternMTFEntry{
				{
					Frequency: "5m",
					Label:     "5m",
					ThreeBar: &slots.KlinePatternThreeBarWindow{
						Bars: []slots.KlinePatternBarWindowEntry{
							{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}}},
						},
					},
				},
				{
					Frequency: "60m",
					Label:     "1小时",
					ThreeBar: &slots.KlinePatternThreeBarWindow{
						Bars: []slots.KlinePatternBarWindowEntry{
							{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{LabelCN: "射击之星", Direction: -1}}},
						},
					},
				},
				{
					Frequency: "daily",
					Label:     "日线",
					ThreeBar: &slots.KlinePatternThreeBarWindow{
						Bars: []slots.KlinePatternBarWindowEntry{
							{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{LabelCN: "十字星", Direction: 2}}},
						},
					},
				},
			},
		},
		KlinePatternSummary: "多周期加权看多（0.20）",
		KlinePatternBias:    &slots.KlinePatternBias{Bias: "看多", Score: 0.2},
	}
	kline := labJEVKlineFromDetail(detail)
	if kline == nil || len(kline.TaggedHits) != 3 {
		t.Fatalf("tagged=%+v", kline)
	}
	state := BuildPriceAnalysisJEVState(pa, "buy", kline)
	for _, want := range []string{
		"5m 锚点命中：锤子 方向=看多",
		"1小时 命中：射击之星 方向=看空",
		"日线 命中：十字星 方向=中性",
		"形态展示摘要（多周期加权，仅供参考）",
	} {
		if !strings.Contains(state, want) {
			t.Fatalf("missing %q in state:\n%s", want, state)
		}
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

func mockDecisionChatResponse(w http.ResponseWriter, content string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    0,
		"message": "ok",
		"data": map[string]any{
			"model":   "jev-1.13.0",
			"content": content,
		},
	})
}

const testLabJEVReasonSummary = "多周期结构与锚点形态同向，短线偏多。"

func jevTestServer(t *testing.T, choice string) *httptest.Server {
	t.Helper()
	return jevTestServerWithReason(t, choice, testLabJEVReasonSummary)
}

func jevTestServerWithReason(t *testing.T, choice, reasonSummary string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
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
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, reasonSummary)
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
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
	if out.RecommendedReason != testLabJEVReasonSummary {
		t.Fatalf("reason=%q want LLM summary", out.RecommendedReason)
	}
	if strings.Contains(out.RecommendedReason, "结构+形态 JEV 倾向") {
		t.Fatal("must not use template recommended_reason")
	}
}

func TestResolvePriceAnalysisJEVReasonSummaryFailureKeepsChoice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "看空",
					"probabilities": map[string]float64{"看多": 0.2, "看空": 0.7, "不明确": 0.1},
					"confidence": 0.7,
				},
			})
		case "/v1/decision/chat":
			w.WriteHeader(http.StatusBadGateway)
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏空"}}}
	out := ResolvePriceAnalysisJEV(context.Background(), jev, pa, "sell", nil)
	if !out.OK || out.RecommendedChoice != "看空" {
		t.Fatalf("choice must stay from JEV: %+v", out)
	}
	if !strings.HasPrefix(out.RecommendedReason, labJEVReasonSummaryFailedPrefix) {
		t.Fatalf("reason=%q", out.RecommendedReason)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "不明确",
					"probabilities": map[string]float64{"看多": 0.2, "看空": 0.2, "不明确": 0.6},
					"confidence":    0.5,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "结构与形态信息均不足，暂无法给出可靠方向。")
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
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
