package chat

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func appendSignalDiagnoseChatBody(b *strings.Builder, flow *Flow) {
	if flow == nil {
		return
	}
	if flow.SignalEval.Method != "" {
		appendSignalEvalHitRateCompact(b, flow.SignalEval)
	}
	if doc := renderSignalDiagnoseChatComposeCards(flow); doc != "" {
		fmt.Fprintf(b, "\n\n%s", doc)
	}
}

func appendSignalEvalHitRateCompact(b *strings.Builder, eval slots.SignalEpisodeEval) {
	fmt.Fprintf(b, "\n### Strict / Path 命中率（摘要 · %s）\n\n", eval.Method)
	fmt.Fprintf(b, "| 方向 | Strict 闭合 | Strict 命中 | Path 段数 | Path 命中 | 均收益(Strict) | 均 peak(Path) |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- | --- | --- |\n")
	writeEvalHitRateCompactRow(b, "买入", eval.BuyEpisodes, eval.BuyPathEpisodes)
	writeEvalHitRateCompactRow(b, "卖出", eval.SellEpisodes, eval.SellPathEpisodes)
}

func writeEvalHitRateCompactRow(b *strings.Builder, label string, strict, path slots.SignalEpisodeMetrics) {
	strictHit := "-"
	pathHit := "-"
	avg := "-"
	peak := "-"
	if strict.CompleteCount > 0 {
		strictHit = fmt.Sprintf("%d/%d · %.0f%%", strict.HitCount, strict.CompleteCount, strict.HitRate*100)
		avg = slots.FormatPct(strict.AvgReturn)
	}
	if path.CompleteCount > 0 {
		pathHit = fmt.Sprintf("%d/%d · %.0f%%", path.HitCount, path.CompleteCount, path.HitRate*100)
		peak = slots.FormatPct(path.AvgPeakReturn)
	}
	fmt.Fprintf(b, "| %s | %d | %s | %d | %s | %s | %s |\n",
		label, strict.CompleteCount, strictHit, path.CompleteCount, pathHit, avg, peak)
}

func renderSignalDiagnoseChatComposeCards(flow *Flow) string {
	bodies := signalDiagnoseChatSectionBodies(flow, flow.EvalJudgment)
	if sectionBodiesEmpty(bodies) {
		return ""
	}
	return llm.RenderComposeSectionDocumentAll(bodies)
}

func sectionBodiesEmpty(bodies [4]string) bool {
	for _, s := range bodies {
		if strings.TrimSpace(s) != "" {
			return false
		}
	}
	return true
}

func signalDiagnoseChatSectionBodies(flow *Flow, judgment string) [4]string {
	judgment = strings.TrimSpace(judgment)
	var fromLLM [4]string
	switch {
	case strings.Contains(judgment, "[[S1]]"):
		fromLLM = llm.ParseComposeSectionBodies(judgment)
	default:
		if bodies, ok := llm.ParseRenderedComposeSectionDocument(judgment); ok {
			fromLLM = bodies
		} else if judgment != "" {
			fromLLM[3] = judgment
		}
	}
	derived := deriveSignalDiagnoseSectionBodies(flow)
	for i := range fromLLM {
		if strings.TrimSpace(fromLLM[i]) == "" {
			fromLLM[i] = derived[i]
		}
	}
	return fromLLM
}

func deriveSignalDiagnoseSectionBodies(flow *Flow) [4]string {
	var bodies [4]string
	bodies[0] = deriveComposeSectionSignalTrigger(flow)
	bodies[1] = deriveComposeSectionProbeHitRate(flow)
	bodies[2] = deriveComposeSectionEpisodeRisk(flow)
	bodies[3] = deriveComposeSectionOverall(flow)
	return bodies
}

func deriveComposeSectionSignalTrigger(flow *Flow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "在 %d 个月、%d 根 K 线窗口内，probe 触发买入 %d 次、卖出 %d 次。",
		flow.MonthsBack, flow.ProbeBarCount, flow.ProbeBuyHits, flow.ProbeSellHits)
	if flow.DiagnoseVerdict != "" || flow.DiagnoseSummary != "" {
		fmt.Fprintf(&b, " diagnose 结论「%s」", flow.DiagnoseVerdict)
		if s := strings.TrimSpace(flow.DiagnoseSummary); s != "" {
			fmt.Fprintf(&b, "：%s", s)
		}
		fmt.Fprintf(&b, "。")
	}
	if n := diagnoseRuleCount(flow.DiagnoseRaw); n > 0 {
		fmt.Fprintf(&b, " 规则明细共 %d 条（完整触发样例见知识库）。", n)
	}
	return strings.TrimSpace(b.String())
}

func deriveComposeSectionProbeHitRate(flow *Flow) string {
	eval := flow.SignalEval
	if eval.Method == "" && flow.ProbeBuyHits+flow.ProbeSellHits == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "probe 买 %d / 卖 %d。", flow.ProbeBuyHits, flow.ProbeSellHits)
	buy, sell := eval.BuyEpisodes, eval.SellEpisodes
	pathBuy, pathSell := eval.BuyPathEpisodes, eval.SellPathEpisodes
	if buy.CompleteCount+sell.CompleteCount == 0 && pathBuy.CompleteCount+pathSell.CompleteCount == 0 {
		b.WriteString(" Episode 可评价样本不足，strict / Path 命中率暂无法计算。")
		return strings.TrimSpace(b.String())
	}
	if buy.CompleteCount > 0 {
		fmt.Fprintf(&b, " Strict 买入 %d 段闭合，命中 %d/%d（%.0f%%，均收益 %s）。",
			buy.CompleteCount, buy.HitCount, buy.CompleteCount, buy.HitRate*100, slots.FormatPct(buy.AvgReturn))
	}
	if sell.CompleteCount > 0 {
		fmt.Fprintf(&b, " Strict 卖出 %d 段闭合，命中 %d/%d（%.0f%%，均收益 %s）。",
			sell.CompleteCount, sell.HitCount, sell.CompleteCount, sell.HitRate*100, slots.FormatPct(sell.AvgReturn))
	}
	if pathBuy.CompleteCount > 0 {
		fmt.Fprintf(&b, " Path 买入 %d 段，命中 %.0f%%（均 peak %s）。",
			pathBuy.CompleteCount, pathBuy.HitRate*100, slots.FormatPct(pathBuy.AvgPeakReturn))
	}
	if pathSell.CompleteCount > 0 {
		fmt.Fprintf(&b, " Path 卖出 %d 段，命中 %.0f%%。",
			pathSell.CompleteCount, pathSell.HitRate*100)
	}
	b.WriteString(" 上方表格为 Strict / Path 摘要。")
	return strings.TrimSpace(b.String())
}

func deriveComposeSectionEpisodeRisk(flow *Flow) string {
	eval := flow.SignalEval
	inc := eval.BuyEpisodes.IncompleteCount + eval.SellEpisodes.IncompleteCount
	var b strings.Builder
	if inc > 0 {
		fmt.Fprintf(&b, "Strict 仍有 %d 段未闭合（买 %d / 卖 %d），解读命中率时需计入尾部风险。",
			inc, eval.BuyEpisodes.IncompleteCount, eval.SellEpisodes.IncompleteCount)
	}
	if eval.BuyPathEpisodes.CompleteCount > 0 && eval.BuyPathEpisodes.AvgMaxDrawdown != 0 {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "Path 买入段均 maxDD（自高点）约 %s。", slots.FormatPct(eval.BuyPathEpisodes.AvgMaxDrawdown))
	}
	if flow.UsePriceAnalysisEpisodeFilter {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString("已开启价格分析 Episode 过滤，部分触发可能被剔除。")
	}
	if b.Len() == 0 {
		b.WriteString("闭合段与 Path 段数量与上方命中率一致；未见额外未闭合或过滤告警。")
	}
	b.WriteString(" 逐条 Episode 明细仅写入知识库。")
	return strings.TrimSpace(b.String())
}

func deriveComposeSectionOverall(flow *Flow) string {
	if flow.ComposeFallback {
		return fallbackSignalDiagnoseJudgment(flow)
	}
	if flow.DiagnoseVerdict != "" {
		return fmt.Sprintf("综合 diagnose 结论「%s」与 Episode 指标，建议结合 Path 命中率与未闭合段情况决策是否调整参数或延长回溯。", flow.DiagnoseVerdict)
	}
	return "请结合四段诊断与 Strict / Path 摘要判断策略在该标的上的可用性；完整方法论与 Episode 表见知识库。"
}
