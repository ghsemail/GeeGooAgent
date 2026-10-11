package strategydev

import (
	"strings"
	"testing"
)

func TestAppendPriceEpisodeFilterMarkdownAllStripped(t *testing.T) {
	pf := &PriceFilterResult{
		Regime: RegimeUnclear, Side: SideNone,
		AllowLong: false, AllowShort: false,
		Notes: []string{"CHOCH：逆结构首次突破（仅预警）"},
	}
	st := &EpisodeTriggerFilterStats{
		PreFilterBuyHits: 2, PreFilterSellHits: 1,
		StrippedBuy: 2, StrippedSell: 1,
		PostFilterBuyHits: 0, PostFilterSellHits: 0,
		InterceptionBasis: BuildPriceFilterInterceptionBasis(*pf),
		StrippedBuyTriggers: []StrippedEpisodeTrigger{
			{Time: "2026-01-01 10:00 (Asia/Shanghai)", Close: 101},
			{Time: "2026-01-02 10:00 (Asia/Shanghai)", Close: 102},
		},
		StrippedSellTriggers: []StrippedEpisodeTrigger{
			{Time: "2026-01-03 10:00 (Asia/Shanghai)", Close: 100},
		},
	}
	var b strings.Builder
	AppendPriceEpisodeFilterMarkdown(&b, pf, st)
	out := b.String()
	for _, want := range []string{
		"剔除明细", "拦截依据", "不明确", "被掩码剔除的买入触发", "被掩码剔除的卖出触发",
		"过滤前 买 2 / 卖 1", "掩码剔除 买 2 / 卖 1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
