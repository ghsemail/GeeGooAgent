package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

func sampleLabTA00700() *LabTechnicalAnalysis {
	return &LabTechnicalAnalysis{
		Code:            "00700.HK",
		StrategyVersion: "alpha",
		OK:              true,
		Timeframes: map[string]LabTechnicalAnalysisTF{
			"daily": {
				Frequency: "daily", OK: true,
				Indicators: signalclient.TechnicalAnalysisIndicators{
					EMA: signalclient.TechnicalAnalysisEMA{
						EMA13Slope: -2.7116, ValueZone: "below_value_area",
					},
					MACD:       signalclient.TechnicalAnalysisMACD{Histogram: -3.4064, HistogramSlope: 0.3271},
					Stochastic: signalclient.TechnicalAnalysisStochastic{Zone: "oversold"},
					RSI:        signalclient.TechnicalAnalysisRSI{Zone: "neutral"},
					ForceIndex: signalclient.TechnicalAnalysisForceIndex{EMA13: -113544869.847},
				},
			},
			"60m": {
				Frequency: "60m", OK: true,
				Indicators: signalclient.TechnicalAnalysisIndicators{
					EMA: signalclient.TechnicalAnalysisEMA{
						EMA13Slope: 0.0474, ValueZone: "inside_value_area",
					},
					MACD: signalclient.TechnicalAnalysisMACD{Histogram: 1.4663, HistogramSlope: -0.0696},
					Stochastic: signalclient.TechnicalAnalysisStochastic{Zone: "neutral"},
					RSI:        signalclient.TechnicalAnalysisRSI{Zone: "neutral"},
					ForceIndex: signalclient.TechnicalAnalysisForceIndex{EMA13: -1821126.9686},
				},
			},
		},
	}
}

func TestBuildTechnicalAnalysisJEVStateTripleScreen(t *testing.T) {
	state := BuildTechnicalAnalysisJEVState(sampleLabTA00700(), "sell", "60m")
	for _, want := range []string{
		"00700.HK",
		"日线（三重滤网方向层）",
		"60m（三重滤网过滤层）",
		"价值区下方",
		"Impulse",
		"fi13=neg",
		"卖出触发",
	} {
		if !strings.Contains(state, want) {
			t.Fatalf("missing %q in state:\n%s", want, state)
		}
	}
}

func TestElderImpulseLabel(t *testing.T) {
	if elderImpulseLabel(1, 1) != "绿" {
		t.Fatal("expected green")
	}
	if elderImpulseLabel(-1, -1) != "蓝" {
		t.Fatal("expected blue")
	}
	if elderImpulseLabel(1, -1) != "红" {
		t.Fatal("expected red conflict")
	}
}

func TestResolveTechnicalAnalysisJEVMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "看空",
					"probabilities": map[string]float64{"看多": 0.02, "看空": 0.93, "不明确": 0.05},
					"confidence": 0.89,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "日线价值区下方且 Force 偏空，60m 过滤不否决。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	out := ResolveTechnicalAnalysisJEV(context.Background(), jev, sampleLabTA00700(), "sell", "60m")
	if !out.OK {
		t.Fatalf("expected ok, err=%q", out.Error)
	}
	if out.RecommendedChoice != "看空" {
		t.Fatalf("choice=%q", out.RecommendedChoice)
	}
	if out.Choices[1].Probability != 0.93 {
		t.Fatalf("probs=%+v", out.Choices)
	}
	if !strings.Contains(out.RecommendedReason, "价值区") {
		t.Fatalf("reason=%q", out.RecommendedReason)
	}
}

func TestResolveTechnicalAnalysisJEVNoFallbackOnSignalError(t *testing.T) {
	ta := &LabTechnicalAnalysis{OK: false, Error: "lab-api: HTTP 503"}
	out := ResolveTechnicalAnalysisJEV(context.Background(), NewJevClient(JevConfig{BaseURL: "http://unused", APIKey: "k"}), ta, "buy", "60m")
	if out.OK {
		t.Fatal("must not invent direction on failed TA")
	}
	if out.RecommendedChoice != "" {
		t.Fatalf("unexpected choice %q", out.RecommendedChoice)
	}
}

func TestFetchLabTechnicalAnalysisMockSignal(t *testing.T) {
	var gotBodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/lab/technical-analysis" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotBodies = append(gotBodies, body)
		freq, _ := body["frequency"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "00700.HK", "frequency": freq, "strategy_version": "alpha", "months_back": body["months_back"],
			"bar_count": 10, "last_bar_time": "2026-10-27 00:00:00",
			"indicators": map[string]any{
				"ema": map[string]any{"ema_13": 1.0, "ema_26": 2.0, "ema_13_slope": -1.0, "ema_26_slope": -1.0, "value_zone": "below_value_area"},
				"envelope": map[string]any{"mid_ema_22": 1.0, "upper": 2.0, "lower": 0.5, "half_width": 0.75},
				"macd": map[string]any{"line": 0.0, "signal": 0.0, "histogram": -1.0, "histogram_slope": -0.1},
				"stochastic": map[string]any{"slow_k": 10.0, "slow_d": 10.0, "zone": "oversold"},
				"rsi": map[string]any{"value": 30.0, "zone": "neutral"},
				"force_index": map[string]any{"ema_2": -1.0, "ema_13": -2.0},
			},
		})
	}))
	defer srv.Close()

	client := signalclient.NewHTTPClient(srv.URL, "key")
	ta := fetchLabTechnicalAnalysis(context.Background(), client, LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 3,
	}, nil)
	if !ta.OK {
		t.Fatalf("expected ok ta, err=%q frames=%+v", ta.Error, ta.Timeframes)
	}
	if len(ta.Timeframes) != 2 {
		t.Fatalf("expected daily+60m, got %+v", ta.Timeframes)
	}
	for _, body := range gotBodies {
		if body["strategy_version"] != "alpha" {
			t.Fatalf("must send English alpha, got %v", body["strategy_version"])
		}
	}
}

func TestBuildLabEpisodeDetailIncludesTechnicalAnalysisJEV(t *testing.T) {
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
			mockDecisionChatResponse(w, "指标偏多。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	bars := []any{map[string]any{"time": float64(1_700_000_000), "close": 50.0}}
	labHTTP := signalclient.NewHTTPClient(srv.URL, "k")
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "daily", Months: 1, Bars: bars,
		LabFetcher: stubLabBarsOK{}, TAClient: labHTTP, Jev: jev,
		JEVTimeout: 3 * time.Second,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.TechnicalAnalysis == nil || !detail.TechnicalAnalysis.OK {
		t.Fatalf("technical_analysis=%+v", detail.TechnicalAnalysis)
	}
	if detail.TechnicalAnalysisJEV == nil || !detail.TechnicalAnalysisJEV.OK {
		t.Fatalf("technical_analysis_jev=%+v", detail.TechnicalAnalysisJEV)
	}
	if jevCalls < 3 {
		t.Fatalf("expected PA + TA + unified JEV calls (kline optional), got %d", jevCalls)
	}
	if detail.PriceAnalysisJEV == nil || !detail.PriceAnalysisJEV.OK {
		t.Fatalf("price_analysis_jev=%+v", detail.PriceAnalysisJEV)
	}
	if detail.JEVUnified == nil {
		t.Fatal("expected jev_unified from unified JEV pass")
	}
}
