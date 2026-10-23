package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestLabModeLightEnrichPopulatesThreeBarOnEpisodes(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	buyMerged := []any{0, 1}
	sellMerged := []any{0, -1}
	hits := []any{
		nil,
		[]any{map[string]any{"name": "ThreeWhiteSoldiers", "direction": 1}},
	}
	flow := &Flow{
		Template: SkillSignalDiagnose, LabMode: true, StockCode: "00700.HK",
		Strategies: []StrategyItem{{Frequency: "60m"}},
		ProbeRaw: map[string]any{
			"bars": bars, "buy_merged": buyMerged, "sell_merged": sellMerged,
			"pattern_hits": hits,
		},
		MonthsBack: 3,
	}
	r := &Runner{}
	if err := r.phaseSignalDiagnoseEvaluate(context.Background(), flow, tools.Context{}, nil); err != nil {
		t.Fatal(err)
	}
	if len(flow.SignalEval.BuyDetails) == 0 {
		t.Fatal("expected buy episode")
	}
	ep := flow.SignalEval.BuyDetails[0]
	if ep.KlinePatternThreeBar == nil {
		t.Fatal("expected kline_pattern_three_bar on episode after light enrich")
	}
	if len(ep.KlinePatterns) == 0 {
		t.Fatal("expected kline_patterns on episode")
	}
	cp := chartProbePayload(flow)
	eps, _ := cp["episodes"].([]map[string]any)
	if len(eps) == 0 {
		t.Fatal("expected chart episodes")
	}
	if eps[0]["kline_pattern_three_bar"] == nil {
		t.Fatal("chart episode overlay missing kline_pattern_three_bar")
	}
}
