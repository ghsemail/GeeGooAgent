package chat

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestCardPayloadOmitsPriceAnalysisWhenMasterSwitchOff(t *testing.T) {
	strategydev.SetPriceAnalysisFeatureEnabledForTest(t, false)
	strategydev.SetKlinePatternDisplayEnabledForTest(t, false)
	pa := strategydev.BuildPriceAnalysisFromBars(map[string][]any{
		"60m": strategydevTestBars(90),
	}, 12.3, strategydev.KeyLevelBands{}, "")
	flow := &Flow{
		Template:                       SkillSignalDiagnose,
		UsePriceAnalysisEpisodeFilter:    true,
		PriceAnalysis:                  &pa,
		PriceEpisodeFilterStats:        &strategydev.EpisodeTriggerFilterStats{BuyTriggers: []strategydev.EpisodeTriggerRecord{{Time: "t"}}},
		PriceFilter:                    &strategydev.PriceFilterResult{Regime: strategydev.RegimeUp},
		ProbeRaw:                       map[string]any{"bars": strategydevTestBars(5), "buy_merged": []any{0}, "sell_merged": []any{0}},
	}
	out := CardPayload(flow)
	for _, key := range []string{"price_analysis", "price_filter", "price_episode_filter_stats", "use_price_analysis_episode_filter"} {
		if _, ok := out[key]; ok {
			t.Fatalf("unexpected %q in card when PA disabled", key)
		}
	}
	cp := chartProbePayload(flow)
	raw, _ := json.Marshal(cp)
	if string(raw) != "" {
		for _, key := range []string{"price_analysis", "price_episode_filter_stats", "buy_triggers"} {
			if cp[key] != nil {
				t.Fatalf("chart_probe must omit %q, got %v", key, cp[key])
			}
		}
	}
}

func TestCardPayloadIncludesKlineTriggersWhenPatternDisplayOnAndPADisabled(t *testing.T) {
	strategydev.SetPriceAnalysisFeatureEnabledForTest(t, false)
	strategydev.SetKlinePatternDisplayEnabledForTest(t, true)
	flow := &Flow{
		Template: SkillSignalDiagnose,
		PriceEpisodeFilterStats: &strategydev.EpisodeTriggerFilterStats{
			BuyTriggers: []strategydev.EpisodeTriggerRecord{{
				Time: "t", KlinePatternSummary: "锤头线(看多)",
			}},
		},
		ProbeRaw: map[string]any{"bars": strategydevTestBars(5), "buy_merged": []any{0}, "sell_merged": []any{0}},
	}
	out := CardPayload(flow)
	if out["buy_triggers"] == nil {
		t.Fatal("expected buy_triggers for kline pattern display")
	}
	if _, ok := out["price_episode_filter_stats"]; ok {
		t.Fatal("price_episode_filter_stats should stay omitted when PA off")
	}
}

func TestCardPayloadIncludesPriceAnalysisWhenMasterSwitchOn(t *testing.T) {
	strategydev.SetPriceAnalysisFeatureEnabledForTest(t, true)
	pa := strategydev.BuildPriceAnalysisFromBars(map[string][]any{
		"60m": strategydevTestBars(90),
	}, 12.3, strategydev.KeyLevelBands{}, "")
	flow := &Flow{
		Template:      SkillSignalDiagnose,
		PriceAnalysis: &pa,
		ProbeRaw:      map[string]any{"bars": strategydevTestBars(5), "buy_merged": []any{0}, "sell_merged": []any{0}},
	}
	out := CardPayload(flow)
	if out["price_analysis"] == nil {
		t.Fatal("expected price_analysis when enabled")
	}
}
