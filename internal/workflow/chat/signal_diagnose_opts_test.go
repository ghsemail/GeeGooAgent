package chat

import "testing"

func TestParseSignalDiagnoseOptsProbeOverrides(t *testing.T) {
	raw := map[string]any{
		"signal_diagnose": map[string]any{
			"use_price_analysis_episode_filter": true,
			"price_analysis_buy_score_direction": "reverse",
			"frequency":                         "60m",
			"months_back":                3,
			"buy_signal": []any{
				map[string]any{"index": "SAR", "type": "signal"},
			},
			"sell_signal": []any{
				map[string]any{"index": "SAR", "type": "signal"},
			},
		},
	}
	o := ParseSignalDiagnoseOptsFromWorkflowOptions(raw)
	if o == nil {
		t.Fatal("nil opts")
	}
	if o.UsePriceAnalysisEpisodeFilter == nil || !*o.UsePriceAnalysisEpisodeFilter {
		t.Fatal("price filter")
	}
	if o.PriceAnalysisBuyScoreDirection != "reverse" {
		t.Fatalf("buy score direction=%q", o.PriceAnalysisBuyScoreDirection)
	}
	if o.Frequency != "60m" || o.MonthsBack != 3 {
		t.Fatalf("freq/months=%q %d", o.Frequency, o.MonthsBack)
	}
	if len(o.BuySignal) != 1 || len(o.SellSignal) != 1 {
		t.Fatalf("buy=%d sell=%d", len(o.BuySignal), len(o.SellSignal))
	}
}
