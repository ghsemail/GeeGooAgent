package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestCardPayloadIncludesPriceAnalysis(t *testing.T) {
	pa := strategydev.BuildPriceAnalysisFromBars(map[string][]any{
		"5m": strategydevTestBars(90), "60m": strategydevTestBars(90),
		"daily": strategydevTestBars(50), "weekly": strategydevTestBars(25),
	}, 12.3, strategydev.KeyLevelBands{}, "")
	flow := &Flow{
		Template:      SkillSignalDiagnose,
		StockCode:     "00772.HK",
		ProbeRaw:      map[string]any{"bars": strategydevTestBars(10), "buy_merged": []any{0}, "sell_merged": []any{0}},
		PriceAnalysis: &pa,
		PriceEpisodeFilterStats: &strategydev.EpisodeTriggerFilterStats{
			BuyTriggers: []strategydev.EpisodeTriggerRecord{{Time: "t", Close: 1}},
		},
	}
	out := CardPayload(flow)
	if out["price_analysis"] == nil {
		t.Fatal("missing price_analysis on card")
	}
	cp := chartProbePayload(flow)
	if cp["price_analysis"] == nil || cp["buy_triggers"] == nil {
		t.Fatalf("chart_probe=%v", cp)
	}
}

func strategydevTestBars(n int) []any {
	bars := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.1
		bars[i] = map[string]any{"high": c + 1, "low": c - 1, "close": c, "volume": 1000.0}
	}
	return bars
}
