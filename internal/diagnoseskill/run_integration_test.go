package diagnoseskill

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func mockMacroDiagnoseServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "kline-pattern"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "00700.HK", "signal_frequency": "60m",
				"kline_pattern": []map[string]any{{
					"frequency": "60m", "time": "2026-11-20 16:00 (Asia/Shanghai)",
					"open": 340.0, "high": 345.0, "low": 338.0, "close": 344.0,
					"hits": []map[string]any{{"name": "Doji", "type": "pattern", "signal": 2}},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "price-analysis"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "00700.HK", "signal_frequency": "60m", "spot_price": 344.0,
				"price_analysis": []map[string]any{{
					"frequency": "60m", "time": "2026-11-20 16:00 (Asia/Shanghai)", "close": 344.0,
					"structure": []string{"LH", "LL"}, "range_position": 0.3,
					"hits": []map[string]any{{"name": "CHOCH", "type": "structure", "signal": -1}},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "technical-analysis"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "00700.HK", "signal_frequency": "60m",
				"technical_analysis": []map[string]any{{
					"frequency": "60m", "time": "2026-11-20 16:00 (Asia/Shanghai)",
					"macd": map[string]any{"histogram": -0.5},
					"hits": []map[string]any{{"name": "MACD", "type": "flag", "signal": -1}},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "index-trend"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "00700.HK", "signal_frequency": "60m",
				"index_trend": []map[string]any{{
					"frequency": "60m",
					"hits": []map[string]any{
						{"name": "ADX", "type": "flag", "signal": -1},
						{"name": "EMA", "type": "flag", "signal": 0},
						{"name": "SAR", "type": "flag", "signal": -1},
					},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func mockJEVServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev-test","answers":{"direction":{"type":"choice","choice":"看空","probabilities":{"看多":0.1,"看空":0.8,"不明确":0.1},"confidence":0.8}}}}`))
		case "/v1/decision/chat":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev-test","content":"依据结构 LH+LL 与 MACD 柱为负，偏空。"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestRunStrategySkillShape(t *testing.T) {
	macroSrv := mockMacroDiagnoseServer(t)
	defer macroSrv.Close()
	jevSrv := mockJEVServer(t)
	defer jevSrv.Close()

	jev := strategydev.NewJevClient(strategydev.JevConfig{BaseURL: jevSrv.URL, APIKey: "k", Model: "jev-test"})
	res, err := Run(context.Background(), RunInput{
		Code: "00700.HK", SignalFreq: "60m",
		Lanes: &LaneClient{BaseURL: macroSrv.URL, APIKey: "macro-key"},
		JEV:   jev,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res["signal_frequency"] != "60m" {
		t.Fatalf("freq=%v", res["signal_frequency"])
	}
	jevU, _ := res["jev_unified"].(UnifiedJEV)
	if jevU.OK && jevU.Direction == "" {
		t.Fatal("expected direction when unified ok")
	}
}

func TestRunRequiresStrategyFrequency(t *testing.T) {
	_, err := Run(context.Background(), RunInput{Code: "00700.HK", Lanes: &LaneClient{BaseURL: "http://127.0.0.1:1"}})
	if err == nil || !strings.Contains(err.Error(), "signal_frequency") {
		t.Fatalf("err=%v", err)
	}
}

func TestMacroResultOmitsSignalFrequency(t *testing.T) {
	macroSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"code": "00700.HK",
			"kline_pattern": []map[string]any{{
				"frequency": "daily", "time": "2026-11-20 00:00 (Asia/Shanghai)",
				"open": 1, "high": 1, "low": 1, "close": 1, "hits": []any{},
			}},
		}
		key := strings.TrimPrefix(r.URL.Path, "/v1/diagnose/macro/")
		switch key {
		case "price-analysis":
			payload["price_analysis"] = []map[string]any{{"frequency": "daily", "structure": []string{"LH"}, "close": 1.0}}
		case "technical-analysis":
			payload["technical_analysis"] = []map[string]any{{"frequency": "daily", "rsi": map[string]any{"value": 40.0}}}
		case "index-trend":
			payload["index_trend"] = []map[string]any{{"frequency": "daily", "hits": []map[string]any{{"name": "ADX", "signal": 0}}}}
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer macroSrv.Close()
	jevSrv := mockJEVServer(t)
	defer jevSrv.Close()
	jev := strategydev.NewJevClient(strategydev.JevConfig{BaseURL: jevSrv.URL, APIKey: "k"})
	res, err := Run(context.Background(), RunInput{
		Macro: true, Code: "00700.HK",
		Lanes: &LaneClient{BaseURL: macroSrv.URL}, JEV: jev,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, has := res["signal_frequency"]; has {
		t.Fatal("macro result must not include signal_frequency")
	}
}