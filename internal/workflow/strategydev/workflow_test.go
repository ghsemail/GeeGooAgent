package strategydev

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestFilterSkipsRangeAndUnclear(t *testing.T) {
	cat := Catalog{Indices: []Candidate{{SignalID: "s1", Name: "MACD金死叉信号", Kind: "index", Buy: []any{map[string]any{"index": "MACD"}}}}}
	buy, sell := FilterCandidates(cat, RegimeRange, SideNone, "60m")
	if len(buy) != 0 || len(sell) != 0 {
		t.Fatalf("expected no candidates for range")
	}
	buy, sell = FilterCandidates(cat, RegimeUnclear, SideNone, "60m")
	if len(buy) != 0 || len(sell) != 0 {
		t.Fatalf("expected no candidates for unclear")
	}
}

func TestDowntrendBuyCannotBeNoSignal(t *testing.T) {
	bad := Candidate{SignalID: "x", Name: "bad", Buy: []any{map[string]any{"index": "nosignal"}}}
	if !isNoSignalRules(bad.Buy) {
		t.Fatal("expected nosignal")
	}
	cat := Catalog{
		Combinations: []Candidate{{
			SignalID: coverComboID,
			Name:     "SAR信号配套 MACD 直方图趋势",
			Kind:     "combination",
			Buy:      []any{map[string]any{"index": "SAR", "type": "signal"}},
		}},
		Indices: []Candidate{{
			SignalID: "662cb6efc4cee7ffb800d0a6",
			Name:     "MACD金死叉信号",
			Kind:     "index",
			Buy:      []any{map[string]any{"index": "MACD", "type": "signal"}},
		}},
	}
	buy, sell := FilterCandidates(cat, RegimeDown, SideRight, "60m")
	if len(sell) == 0 {
		t.Fatal("expected sell candidate")
	}
	if len(buy) == 0 || isNoSignalRules(buy[0].Buy) {
		t.Fatal("expected real cover buy")
	}
	res, err := Run(context.Background(), Input{Code: "00700.HK", Frequency: "60m"}, Deps{
		Run: mockRunner(t, cat, buy[0], sell[0]),
		Jev: NewJevClient(JevConfig{}),
		Now: func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.SelectedBuy == nil || isNoSignalRules(res.SelectedBuy.Buy) {
		t.Fatal("selected buy must not be nosignal")
	}
}

func TestHardVerdictBlocksPass(t *testing.T) {
	if ApplyHardVerdictRules(DiagnoseSummary{CompleteSegs: 2, AvgDirection: 0.05}) != VerdictLongWindow {
		t.Fatal("complete<3 must not pass")
	}
	if ApplyHardVerdictRules(DiagnoseSummary{CompleteSegs: 4, AvgDirection: 0}) != VerdictSwapPair {
		t.Fatal("non-positive avg must not pass")
	}
	if ApplyHardVerdictRules(DiagnoseSummary{CompleteSegs: 3, AvgDirection: 0.01}) != VerdictPass {
		t.Fatal("expected pass")
	}
}

func TestClassifyDownTrend(t *testing.T) {
	n := 30
	bars := make([]any, n)
	series := map[string][]float64{
		"ADX":     make([]float64, n),
		"PLUS_DI": make([]float64, n),
		"MINUS_DI": make([]float64, n),
	}
	for i := 0; i < n; i++ {
		bars[i] = map[string]any{"close": 100.0, "high": 101.0, "low": 99.0, "volume": 1000.0}
		series["ADX"][i] = 30
		series["PLUS_DI"][i] = 10
		series["MINUS_DI"][i] = 25
	}
	ro := ClassifyRegime(bars, series, nil)
	if ro.Regime != RegimeDown || ro.Side != SideRight {
		t.Fatalf("got %s %s", ro.Regime, ro.Side)
	}
}

func mockRunner(t *testing.T, cat Catalog, buy, sell Candidate) ToolRunner {
	t.Helper()
	return func(ctx context.Context, name string, args map[string]any) tools.Result {
		switch name {
		case "probe_bot_signal_series":
			n := 60
			bars := make([]any, n)
			for i := 0; i < n; i++ {
				bars[i] = map[string]any{
					"time": "2026-01-01T00:00:00Z", "close": 100.0,
					"high": 101.0, "low": 99.0, "volume": 1000.0,
				}
			}
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{
				"bars": bars,
				"buy_rules": []any{
					map[string]any{
						"index": "ADX",
						"value_series": map[string]any{
							"ADX":      repeatFloat(30, n),
							"PLUS_DI":  repeatFloat(10, n),
							"MINUS_DI": repeatFloat(25, n),
						},
					},
				},
			}}
		case "get_index_signals":
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
				map[string]any{
					"signal_id": sell.SignalID, "name": sell.Name,
					"buy_signal": sell.Buy, "frequency": "60m",
				},
			}}}
		case "get_signal_combinations":
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
				map[string]any{
					"signal_id": buy.SignalID, "name": buy.Name,
					"buy_signal": buy.Buy, "sell_signal": []any{},
				},
			}}}
		case "get_agent_signal_combination", "get_custom_signal_for_skill":
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{}}}
		case "diagnose_bot_signal_series":
			n := 80
			bars, buyM, sellM := mockMergedSeries(n)
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{
				"bars": bars, "buy_merged": buyM, "sell_merged": sellM,
				"buy_hits": 4, "sell_hits": 4,
			}}
		default:
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{}}
		}
	}
}

func repeatFloat(v float64, n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func mockMergedSeries(n int) ([]any, []any, []any) {
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		bars[i] = map[string]any{"close": 100.0 + float64(i%5), "high": 101.0, "low": 99.0, "volume": 1.0}
		buyM[i] = 0
		sellM[i] = 0
	}
	// three complete short-friendly cycles
	for _, pair := range [][2]int{{10, 20}, {30, 40}, {50, 60}} {
		sellM[pair[0]] = -1
		buyM[pair[1]] = 1
	}
	return bars, buyM, sellM
}
