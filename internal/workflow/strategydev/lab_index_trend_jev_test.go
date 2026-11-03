package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

func sampleLabIndexTrend00700() *LabIndexTrend {
	return &LabIndexTrend{
		Code:            "00700.HK",
		StrategyVersion: "alpha",
		OK:              true,
		Timeframes: map[string]LabIndexTrendTF{
			"daily": {
				Frequency: "daily", OK: true,
				Trends: []signalclient.IndexTrendTrendRow{
					{Name: "ADX趋势", Index: "ADX", Stance: "short", Value: -1},
					{Name: "EMA趋势(中长期)", Index: "EMA", Stance: "short", Value: -1},
					{Name: "SAR抛物线趋势", Index: "SAR", Stance: "flat", Value: 0},
				},
				Summary: signalclient.IndexTrendSummary{Long: 0, Short: 2, Flat: 1, Net: -2},
			},
			"60m": {
				Frequency: "60m", OK: true,
				Trends: []signalclient.IndexTrendTrendRow{
					{Name: "ADX趋势", Index: "ADX", Stance: "long", Value: 1},
					{Name: "EMA趋势(中长期)", Index: "EMA", Stance: "long", Value: 1},
					{Name: "SAR抛物线趋势", Index: "SAR", Stance: "long", Value: 1},
				},
				Summary: signalclient.IndexTrendSummary{Long: 3, Short: 0, Flat: 0, Net: 3},
			},
		},
	}
}

func TestBuildIndexTrendJEVStateMultiTF(t *testing.T) {
	state := BuildIndexTrendJEVState(sampleLabIndexTrend00700(), "sell", "60m", false)
	for _, want := range []string{
		"00700.HK",
		"日线（方向层）",
		"60m（信号周期）",
		"ADX趋势",
		"net=-2",
		"net=3",
		"卖出触发",
	} {
		if !strings.Contains(state, want) {
			t.Fatalf("missing %q in state:\n%s", want, state)
		}
	}
}

func TestResolveIndexTrendJEVMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type": "choice", "choice": "看空",
					"probabilities": map[string]float64{"看多": 0.05, "看空": 0.88, "不明确": 0.07},
					"confidence": 0.88,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "日线三旗偏空 net=-2，信号周期虽多但 JEV 决策权衡偏空。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	out := ResolveIndexTrendJEV(context.Background(), jev, sampleLabIndexTrend00700(), "sell", "60m", false)
	if !out.OK {
		t.Fatalf("expected ok, err=%q", out.Error)
	}
	if out.RecommendedChoice != "看空" {
		t.Fatalf("choice=%q", out.RecommendedChoice)
	}
	if !strings.Contains(out.RecommendedReason, "net") && !strings.Contains(out.RecommendedReason, "偏空") {
		t.Fatalf("reason=%q", out.RecommendedReason)
	}
}

func TestResolveIndexTrendJEVNoFallbackOnSignalError(t *testing.T) {
	it := &LabIndexTrend{OK: false, Error: "lab-api: HTTP 503"}
	out := ResolveIndexTrendJEV(context.Background(), NewJevClient(JevConfig{BaseURL: "http://unused", APIKey: "k"}), it, "buy", "60m", false)
	if out.OK {
		t.Fatal("must not invent direction on failed index-trend")
	}
	if out.RecommendedChoice != "" {
		t.Fatalf("unexpected choice %q", out.RecommendedChoice)
	}
}

func TestFetchLabIndexTrendMockSignal(t *testing.T) {
	var gotBodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/lab/index-trend" {
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
			"trends": []map[string]any{
				{"name": "ADX趋势", "index": "ADX", "type": "flag", "stance": "long", "value": 1},
			},
			"summary": map[string]any{"long": 1, "short": 0, "flat": 2, "net": 1},
		})
	}))
	defer srv.Close()

	client := signalclient.NewHTTPClient(srv.URL, "key")
	it := fetchLabIndexTrend(context.Background(), client, LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 3,
	}, nil)
	if !it.OK {
		t.Fatalf("expected ok, err=%q frames=%+v", it.Error, it.Timeframes)
	}
	if len(it.Timeframes) != 2 {
		t.Fatalf("expected daily+60m, got %+v", it.Timeframes)
	}
	for _, body := range gotBodies {
		if body["strategy_version"] != "alpha" {
			t.Fatalf("must send English alpha, got %v", body["strategy_version"])
		}
	}
}

func TestBuildLabEpisodeDetailIncludesIndexTrendJEV(t *testing.T) {
	var jevCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/lab/index-trend":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "X", "frequency": "daily", "strategy_version": "alpha", "months_back": 3, "bar_count": 5,
				"trends": []map[string]any{
					{"name": "ADX趋势", "index": "ADX", "stance": "long", "value": 1},
				},
				"summary": map[string]any{"long": 2, "short": 0, "flat": 1, "net": 2},
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
			mockDecisionChatResponse(w, "三旗偏多。")
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
		LabFetcher: stubLabBarsOK{}, IndexTrendClient: labHTTP, Jev: jev,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.IndexTrend == nil || !detail.IndexTrend.OK {
		t.Fatalf("index_trend=%+v", detail.IndexTrend)
	}
	if detail.IndexTrendJEV == nil || !detail.IndexTrendJEV.OK {
		t.Fatalf("index_trend_jev=%+v", detail.IndexTrendJEV)
	}
	if jevCalls < 2 {
		t.Fatalf("expected PA + index + unified JEV calls, got %d", jevCalls)
	}
}
