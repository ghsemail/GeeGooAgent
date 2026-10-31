package strategydev

import (
	"fmt"
	"strings"
)

// BuildPriceFilterInterceptionBasis returns a Chinese explanation for why triggers were masked.
func BuildPriceFilterInterceptionBasis(pf PriceFilterResult) string {
	var parts []string
	if !pf.AllowLong && !pf.AllowShort {
		switch pf.Regime {
		case RegimeUnclear:
			parts = append(parts, "价格分析结论为不明确，不允许买入/卖出，故掩码剔除")
		case RegimeRange:
			parts = append(parts, "价格分析结论为横盘，不允许买入/卖出，故掩码剔除")
		default:
			parts = append(parts, fmt.Sprintf("价格分析结论为 %s，不允许买入/卖出，故掩码剔除", pf.Regime))
		}
	} else {
		if !pf.AllowLong {
			parts = append(parts, fmt.Sprintf("价格分析不允许买入（regime=%s），故掩码剔除买入触发", pf.Regime))
		}
		if !pf.AllowShort {
			parts = append(parts, fmt.Sprintf("价格分析不允许卖出（regime=%s），故掩码剔除卖出触发", pf.Regime))
		}
	}
	if len(parts) == 0 {
		return "价格分析允许当前方向，未因方向门控掩码剔除触发"
	}
	return strings.Join(parts, "；")
}

// AppendPriceEpisodeFilterMarkdown writes the full stripped-trigger section for diagnose reports.
// When longOnly is true, sell-side trigger ledgers are omitted (probe 卖次为平多信号，非做空 Episode).
func AppendPriceEpisodeFilterMarkdown(b *strings.Builder, pf *PriceFilterResult, stats *EpisodeTriggerFilterStats, longOnly bool) {
	if pf == nil || stats == nil {
		return
	}
	if stats.StrippedBuy == 0 && stats.StrippedSell == 0 {
		return
	}
	fmt.Fprintf(b, "\n### 价格分析 Episode 过滤 · 剔除明细\n\n")
	fmt.Fprintf(b, "- **结构结论**：%s · %s\n", pf.Regime, pf.Side)
	fmt.Fprintf(b, "- **方向门控**：allow_long=%v · allow_short=%v\n", pf.AllowLong, pf.AllowShort)
	if pf.VolumeSummary != "" {
		fmt.Fprintf(b, "- **量能**：%s\n", pf.VolumeSummary)
	}
	if pf.RVOL > 0 {
		fmt.Fprintf(b, "- **RVOL**：%.2f\n", pf.RVOL)
	}
	if pf.ER > 0 {
		fmt.Fprintf(b, "- **ER**：%.2f\n", pf.ER)
	}
	if len(pf.Notes) > 0 {
		fmt.Fprintf(b, "- **结构附注**：%s\n", strings.Join(pf.Notes, "；"))
	}
	if basis := strings.TrimSpace(stats.InterceptionBasis); basis != "" {
		fmt.Fprintf(b, "- **拦截依据**：%s\n", basis)
	}
	if longOnly {
		fmt.Fprintf(b, "- **触发计数（只做多）**：过滤前 买 %d → 掩码剔除 买 %d → 过滤后 买 %d",
			stats.PreFilterBuyHits, stats.StrippedBuy, stats.PostFilterBuyHits)
		if stats.PostFilterBuyEpisodes > 0 || stats.PostFilterBuyEpisodesComplete > 0 {
			fmt.Fprintf(b, " → Episode 买 %d 段（闭合 %d）",
				stats.PostFilterBuyEpisodes, stats.PostFilterBuyEpisodesComplete)
		}
	} else {
		fmt.Fprintf(b, "- **触发计数**：过滤前 买 %d / 卖 %d → 掩码剔除 买 %d / 卖 %d → 过滤后 买 %d / 卖 %d",
			stats.PreFilterBuyHits, stats.PreFilterSellHits,
			stats.StrippedBuy, stats.StrippedSell,
			stats.PostFilterBuyHits, stats.PostFilterSellHits)
		if stats.PostFilterBuyEpisodes > 0 || stats.PostFilterSellEpisodes > 0 ||
			stats.PostFilterBuyEpisodesComplete > 0 || stats.PostFilterSellEpisodesComplete > 0 {
			fmt.Fprintf(b, " → Episode 买 %d 段（闭合 %d）/ 卖 %d 段（闭合 %d）",
				stats.PostFilterBuyEpisodes, stats.PostFilterBuyEpisodesComplete,
				stats.PostFilterSellEpisodes, stats.PostFilterSellEpisodesComplete)
		}
	}
	fmt.Fprintf(b, "\n")
	appendStrippedTriggerList(b, "被掩码剔除的买入触发", stats.StrippedBuyTriggers)
	if !longOnly {
		appendStrippedTriggerList(b, "被掩码剔除的卖出触发", stats.StrippedSellTriggers)
	}
	appendEpisodeTriggerRecords(b, "全部买入触发（含掩码）", stats.BuyTriggers)
	if !longOnly {
		appendEpisodeTriggerRecords(b, "全部卖出触发（含掩码）", stats.SellTriggers)
	}
}

func appendEpisodeTriggerRecords(b *strings.Builder, title string, items []EpisodeTriggerRecord) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n**%s**（共 %d 个）：\n\n", title, len(items))
	for i, tr := range items {
		flag := ""
		if tr.Masked {
			flag = " · **已掩码**"
		}
		patternNote := ""
		if tr.KlinePatternSummary != "" {
			patternNote = " · " + tr.KlinePatternSummary
		}
		fmt.Fprintf(b, "%d. %s · 收盘 %.4f%s%s\n", i+1, tr.Time, tr.Close, flag, patternNote)
	}
}

func appendStrippedTriggerList(b *strings.Builder, title string, items []EpisodeTriggerRecord) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n**%s**（共 %d 个）：\n\n", title, len(items))
	for i, tr := range items {
		fmt.Fprintf(b, "%d. %s · 收盘 %.4f\n", i+1, tr.Time, tr.Close)
	}
}
