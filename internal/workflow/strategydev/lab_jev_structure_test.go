package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestLabDetailJEVPipelineFourChoiceCalls(t *testing.T) {
	var jevCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			jevCalls++
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "看多",
					"probabilities": map[string]float64{"看多": 0.9, "看空": 0.05, "不明确": 0.05},
					"confidence": 0.9,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "依据充分。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	ctx := context.Background()
	detail := LabEpisodeDetail{
		PriceAnalysis: &PriceAnalysis{
			Code: "00700.HK",
			Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多", Structure: "HL"}},
		},
		KlinePatternSummary: "锤子(看多)",
		KlinePatternBias:    &slots.KlinePatternBias{Bias: "看多"},
		KlinePatterns:       []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
		TechnicalAnalysis:   sampleLabTA00700(),
	}
	in := LabDetailInput{Jev: jev, TAClient: signalclient.NewHTTPClient("http://unused", "k")}

	applyPriceAnalysisJEV(ctx, &detail, "buy", jev)
	applyKlineJEV(ctx, &detail, "buy", jev)
	rec := ResolveTechnicalAnalysisJEV(ctx, jev, detail.TechnicalAnalysis, "buy", "60m")
	detail.TechnicalAnalysisJEV = &rec
	applyLabJEVUnified(ctx, &detail, "buy", in)

	if jevCalls != 4 {
		t.Fatalf("expected 4 JEV choice calls, got %d", jevCalls)
	}
	for name, ok := range map[string]bool{
		"price_analysis_jev":     detail.PriceAnalysisJEV != nil && detail.PriceAnalysisJEV.OK,
		"kline_jev":              detail.KlineJEV != nil && detail.KlineJEV.OK,
		"technical_analysis_jev": detail.TechnicalAnalysisJEV != nil && detail.TechnicalAnalysisJEV.OK,
		"jev_unified":            detail.JEVUnified != nil && detail.JEVUnified.OK,
	} {
		if !ok {
			t.Fatalf("missing or failed %s", name)
		}
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"price_analysis_jev", "kline_jev", "technical_analysis_jev", "jev_unified"} {
		if !jsonContainsKey(raw, key) {
			t.Fatalf("JSON missing %q", key)
		}
	}
}

func TestBuildLabEpisodeDetailThreeJEVWhenNoKline(t *testing.T) {
	var jevCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/lab/technical-analysis":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "X", "frequency": "daily", "strategy_version": "alpha", "months_back": 3, "bar_count": 5,
				"indicators": map[string]any{
					"ema": map[string]any{"value_zone": "inside_value_area"},
					"envelope": map[string]any{}, "macd": map[string]any{"histogram_slope": 0.1},
					"stochastic": map[string]any{"zone": "neutral"}, "rsi": map[string]any{"zone": "neutral"},
					"force_index": map[string]any{"ema_13": 1.0},
				},
			})
		case "/v1/decision/choice":
			jevCalls++
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "看多",
					"probabilities": map[string]float64{"看多": 0.9, "看空": 0.05, "不明确": 0.05},
					"confidence": 0.9,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "依据充分。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	bars := []any{map[string]any{"time": float64(1_700_000_000), "close": 50.0}}
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "daily", Months: 1, Bars: bars,
		LabFetcher: stubLabBarsOK{},
		TAClient:   signalclient.NewHTTPClient(srv.URL, "k"),
		Jev:        NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"}),
		JEVTimeout: 5 * time.Second,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.KlineJEV != nil {
		t.Fatalf("expected no kline_jev without pattern context, got %+v", detail.KlineJEV)
	}
	if jevCalls != 3 {
		t.Fatalf("expected PA + TA + unified JEV calls, got %d", jevCalls)
	}
}

func jsonContainsKey(raw []byte, key string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
