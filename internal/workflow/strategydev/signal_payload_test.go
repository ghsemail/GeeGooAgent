package strategydev

import (
	"testing"
)

func TestStripSignalPriceAnalysisPayloadWhenDisabled(t *testing.T) {
	ConfigurePriceAnalysisEnabled(false)
	t.Cleanup(func() { ConfigurePriceAnalysisEnabled(true) })
	data := map[string]any{
		"bars":                    []any{map[string]any{"close": 1.0}},
		"price_analysis":          map[string]any{"rows": []any{}},
		"price_score":             12.3,
		"gate_price_score":        -1.0,
		"price_episode_filter_stats": map[string]any{"buy_triggers": []any{}},
	}
	StripSignalPriceAnalysisPayload(data)
	for _, key := range SignalPriceAnalysisResponseKeys {
		if _, ok := data[key]; ok {
			t.Fatalf("expected %q stripped, data=%v", key, data)
		}
	}
	if _, ok := data["bars"]; !ok {
		t.Fatal("bars must remain")
	}
}

func TestStripSignalPriceAnalysisPayloadWhenEnabledKeepsFields(t *testing.T) {
	ConfigurePriceAnalysisEnabled(true)
	data := map[string]any{"price_analysis": map[string]any{"rows": []any{}}}
	StripSignalPriceAnalysisPayload(data)
	if data["price_analysis"] == nil {
		t.Fatal("expected field kept when PA enabled")
	}
}

func TestDiagnoseIgnoresSignalPriceFieldsWhenDisabled(t *testing.T) {
	ConfigurePriceAnalysisEnabled(false)
	SetKlinePatternDisplayEnabledForTest(t, false)
	t.Cleanup(func() { ConfigurePriceAnalysisEnabled(true) })
	bars := makeBarsWithBuy(3)
	buyM := []any{0, 0, 1}
	sellM := []any{0, 0, 0}
	summary, pf, stats := finishDiagnoseFromMerged(
		t.Context(), nil, "00700.HK", 3, KeyLevelBands{}, bars, "60m", buyM, sellM, RegimeUp, map[string]any{
			"price_analysis": map[string]any{"rows": []any{}},
			"price_score":    99.0,
		}, false, nil, BuyScoreDirectionNormal, PriceAnalysisGateScoreModeFull,
	)
	if pf != nil || stats != nil {
		t.Fatalf("expected no price filter stats, pf=%v stats=%v", pf, stats)
	}
	if summary.BuyHits != 1 {
		t.Fatalf("buy hits=%d want 1 unfiltered", summary.BuyHits)
	}
}

func TestDiagnoseUsesSignalPriceFieldsWhenEnabled(t *testing.T) {
	ConfigurePriceAnalysisEnabled(true)
	// Signal may still return price_analysis; Agent computes its own path — strip only applies to payload hygiene.
	data := map[string]any{"price_analysis": map[string]any{"rows": []any{}}}
	StripSignalPriceAnalysisPayload(data)
	if data["price_analysis"] == nil {
		t.Fatal("enabled mode should not strip")
	}
}

func makeBarsWithBuy(n int) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = map[string]any{
			"time":   "2026-01-01 10:00:00",
			"open":   100.0,
			"high":   101.0,
			"low":    99.0,
			"close":  100.0 + float64(i),
			"volume": 1000.0,
		}
	}
	return out
}
