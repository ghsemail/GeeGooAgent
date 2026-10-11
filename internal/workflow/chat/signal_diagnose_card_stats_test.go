package chat

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestCardPayloadPriceEpisodeFilterStatsShape(t *testing.T) {
	pa := strategydev.BuildPriceAnalysisFromBars(map[string][]any{
		"5m": strategydevTestBars(90), "60m": strategydevTestBars(90),
		"daily": strategydevTestBars(50), "weekly": strategydevTestBars(25),
	}, 12.3, strategydev.KeyLevelBands{})
	pa.AnchorTime = "2026-03-06 16:00 (Asia/Shanghai)"
	pa.AnchorClose = 12.0
	st := &strategydev.EpisodeTriggerFilterStats{
		BuyTriggers: []strategydev.EpisodeTriggerRecord{{
			Time: "2026-03-09 10:00 (Asia/Shanghai)", Close: 12.5, Masked: true, Intercepted: true,
			PriceAnalysis: &pa, AIInterceptConclusion: "已拦截买入信号：结构不明确", AIInterceptConclusionSource: "rule",
		}},
		StrippedBuyTriggers: []strategydev.EpisodeTriggerRecord{{
			Time: "2026-03-09 10:00 (Asia/Shanghai)", Close: 12.5, Masked: true, Intercepted: true,
			PriceAnalysis: &pa, AIInterceptConclusion: "已拦截买入信号：结构不明确", AIInterceptConclusionSource: "rule",
		}},
	}
	flow := &Flow{
		Template:                      SkillSignalDiagnose,
		UsePriceAnalysisEpisodeFilter: true,
		PriceEpisodeFilterStats:       st,
		PriceAnalysis:                 &pa,
		ProbeRaw: map[string]any{
			"bars": []any{}, "buy_merged": []any{}, "sell_merged": []any{},
		},
	}
	out := CardPayload(flow)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["price_episode_filter_stats"] == nil {
		t.Fatalf("missing stats: %v", m)
	}
	cp := chartProbePayload(flow)
	if cp["price_episode_filter_stats"] == nil {
		t.Fatal("chart_probe missing nested stats")
	}
}
