package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Runner) phaseSignalDiagnoseCompose(
	ctx context.Context,
	flow *Flow,
	_ tools.Context,
	_ func(name, status, summary string),
) error {
	judgment, err := r.synthesizeSignalDiagnoseJudgment(ctx, flow)
	if err != nil {
		return fmt.Errorf("信号评价合成失败：%w", err)
	}
	flow.EvalJudgment = judgment
	flow.KBDraft = assembleSignalDiagnoseDoc(flow, judgment)
	if skipSignalDiagnoseKB() {
		flow.Phase = PhaseSummarize
	} else {
		flow.Phase = PhaseDiagSaveKB
	}
	flow.touch()
	return nil
}

func skipSignalDiagnoseKB() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_SKIP_KB")))
	return v == "1" || v == "true" || v == "yes"
}

func (r *Runner) phaseSignalDiagnoseSaveKB(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	content := strings.TrimSpace(flow.KBDraft)
	if content == "" {
		return terminalError("缺少诊断报告正文")
	}
	label := strategyDisplayLabel(flow)
	stock := coalesceStrings(flow.StockName, flow.StockQuery)
	title := signalDiagnoseKnowledgeTitle(label, stock)
	res := r.runTool(ctx, toolCtx, "save_strategy_knowledge", map[string]any{
		"strategy_name": title,
		"title":         title,
		"content":       content,
		"folder_path":   tools.SignalDiagnoseFolder,
	}, recordTool)
	if res.Status != tools.StatusOK {
		flow.KBDraft = strings.TrimSpace(flow.KBDraft) + fmt.Sprintf(
			"\n\n> ⚠ 知识库写入失败：%s（报告内容仍可在对话中查看）",
			strings.TrimSpace(res.Summary),
		)
	} else {
		flow.KnowledgeID = strings.TrimSpace(fmt.Sprint(res.Data["knowledge_id"]))
		flow.KnowledgeTitle = strings.TrimSpace(fmt.Sprint(res.Data["title"]))
	}
	flow.Phase = PhaseSummarize
	flow.touch()
	return nil
}

func signalDiagnoseKnowledgeTitle(strategy, stock string) string {
	strategy = strings.TrimSpace(strategy)
	stock = strings.TrimSpace(stock)
	switch {
	case strategy != "" && stock != "":
		return fmt.Sprintf("%s · %s · 信号诊断", strategy, stock)
	case strategy != "":
		return fmt.Sprintf("%s · 信号诊断", strategy)
	default:
		return "信号诊断"
	}
}

func (r *Runner) synthesizeSignalDiagnoseJudgment(ctx context.Context, flow *Flow) (string, error) {
	label := strategyDisplayLabel(flow)
	freq := "-"
	if len(flow.Strategies) > 0 && flow.Strategies[0].Frequency != "" {
		freq = flow.Strategies[0].Frequency
	}
	user := fmt.Sprintf(`请根据以下信号诊断与 Episode 评价指标，用 3–5 句中文给出总结判断：

策略：%s（%s）
标的：%s %s
周期 / 回溯：%s / %d 月
probe：买 %d 次 / 卖 %d 次（%d 根 K 线）
diagnose：%s — %s

Episode 评价（strict=至反向信号；path=无反向则至下一同向或样本末，含 peak/maxDD）：
%s

要求：
1. 判断买卖信号整体趋势是否与策略逻辑一致
2. 指出买/卖哪一侧更可靠（若有）
3. 说明样本是否充足、未闭合段是否影响结论
4. 禁止编造未提供的数字；百分比与次数必须与上文一致`,
		label, flow.CatalogType,
		flow.StockName, flow.StockCode,
		freq, flow.MonthsBack,
		flow.ProbeBuyHits, flow.ProbeSellHits, flow.ProbeBarCount,
		flow.DiagnoseVerdict, flow.DiagnoseSummary,
		slots.FormatEvalMetricsJSON(flow.SignalEval),
	)
	if r != nil && r.ComposeLLM != nil {
		resp, err := r.ComposeLLM.Chat(ctx, []llm.Message{
			{Role: llm.RoleSystem, Content: signalDiagnoseJudgmentSystemPrompt()},
			{Role: llm.RoleUser, Content: user},
		}, nil, 0, 0)
		if err != nil {
			return "", err
		}
		if text := strings.TrimSpace(llm.VisibleAssistantContent(resp.Content, resp.ReasoningContent)); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("LLM 返回空内容")
	}
	return fallbackSignalDiagnoseJudgment(flow), nil
}

func signalDiagnoseJudgmentSystemPrompt() string {
	return "你是 GeeGoo 量化策略诊断助手。根据结构化 probe/diagnose/Episode 指标写简短中文评价，不夸大、不编造数据。"
}

func fallbackSignalDiagnoseJudgment(flow *Flow) string {
	buy := flow.SignalEval.BuyEpisodes
	sell := flow.SignalEval.SellEpisodes
	var b strings.Builder
	fmt.Fprintf(&b, "基于 %d 个月 probe 与 Episode 评价（至反向信号）：", flow.MonthsBack)
	pathBuy := flow.SignalEval.BuyPathEpisodes
	pathSell := flow.SignalEval.SellPathEpisodes
	if buy.CompleteCount+sell.CompleteCount == 0 {
		if pathBuy.CompleteCount+pathSell.CompleteCount == 0 {
			b.WriteString("可评价样本不足，建议延长回溯或换周期后再评。")
			return strings.TrimSpace(b.String())
		}
		fmt.Fprintf(&b, " strict 无闭合段（常见：仅买不卖）。")
	}
	if buy.CompleteCount > 0 {
		fmt.Fprintf(&b, " 买入 %d 段闭合 episode 命中率 %.0f%%（均收益 %s）。",
			buy.CompleteCount, buy.HitRate*100, slots.FormatPct(buy.AvgReturn))
	}
	if sell.CompleteCount > 0 {
		fmt.Fprintf(&b, " 卖出 %d 段闭合 episode 命中率 %.0f%%（均收益 %s）。",
			sell.CompleteCount, sell.HitRate*100, slots.FormatPct(sell.AvgReturn))
	}
	if buy.IncompleteCount+sell.IncompleteCount > 0 {
		fmt.Fprintf(&b, " strict 另有 %d 段未闭合。", buy.IncompleteCount+sell.IncompleteCount)
	}
	if pathBuy.CompleteCount > 0 {
		fmt.Fprintf(&b, " Path 买入 %d 段命中率 %.0f%%（均 peak %s，均 maxDD %s）。",
			pathBuy.CompleteCount, pathBuy.HitRate*100, slots.FormatPct(pathBuy.AvgPeakReturn), slots.FormatPct(pathBuy.AvgMaxDrawdown))
	}
	if pathSell.CompleteCount > 0 {
		fmt.Fprintf(&b, " Path 卖出 %d 段命中率 %.0f%%。",
			pathSell.CompleteCount, pathSell.HitRate*100)
	}
	b.WriteString(" 未配置 LLM 时为规则摘要，完整判断请配置 ComposeLLM。")
	return strings.TrimSpace(b.String())
}

func assembleSignalDiagnoseDoc(flow *Flow, judgment string) string {
	var b strings.Builder
	label := strategyDisplayLabel(flow)
	title := FormatSignalDiagnoseMessage(label, coalesceStrings(flow.StockName, flow.StockQuery))
	fmt.Fprintf(&b, "## %s · 信号诊断报告\n\n", title)
	fmt.Fprintf(&b, "_生成时间：%s UTC_\n\n", time.Now().UTC().Format(time.RFC3339))
	b.WriteString(renderSignalDiagnoseFacts(flow))
	appendSignalEvalMethodology(&b)
	appendKlineOverviewSection(&b, flow)
	appendSignalEvalSection(&b, flow.SignalEval)
	appendEpisodeDetailSections(&b, flow.SignalEval)
	fmt.Fprintf(&b, "\n### Agent 评价\n\n%s\n", strings.TrimSpace(judgment))
	appendDiagnoseRuleDetails(&b, flow.DiagnoseRaw)
	if skipSignalDiagnoseKB() {
		fmt.Fprintf(&b, "\n> Workflow：read_strategy → probe → evaluate_accuracy → diagnose → LLM 评价（本次未写入知识库）。\n")
	} else {
		fmt.Fprintf(&b, "\n> Workflow：read_strategy → probe → evaluate_accuracy → diagnose → LLM 评价 → 写入知识库（%s）。\n",
			tools.SignalDiagnoseFolder)
	}
	return strings.TrimSpace(b.String())
}

func renderSignalDiagnoseFacts(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	var b strings.Builder
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 读取策略 | %s（%s） |\n", label, flow.CatalogType)
	fmt.Fprintf(&b, "| 标的 | %s %s |\n", flow.StockName, flow.StockCode)
	freq := "-"
	if len(flow.Strategies) > 0 && flow.Strategies[0].Frequency != "" {
		freq = flow.Strategies[0].Frequency
	}
	fmt.Fprintf(&b, "| 周期 / 回溯 | %s / %d 月 |\n", freq, flow.MonthsBack)
	fmt.Fprintf(&b, "| 信号测试 probe | 买入 %d 次 / 卖出 %d 次", flow.ProbeBuyHits, flow.ProbeSellHits)
	if flow.ProbeBarCount > 0 {
		fmt.Fprintf(&b, "（%d 根 K 线）", flow.ProbeBarCount)
	}
	fmt.Fprintf(&b, " |\n")
	if flow.UseKeyLevelEpisodeStop {
		note := "开启 · 买段跌破支撑 low / 卖段突破阻力 high 提前结束 Strict"
		if kl := flow.KeyLevels; kl.SupportLow > 0 || kl.ResistHigh > 0 {
			note = fmt.Sprintf("%s（支撑 %.2f~%.2f · 阻力 %.2f~%.2f）",
				note, kl.SupportLow, kl.SupportHigh, kl.ResistLow, kl.ResistHigh)
		}
		fmt.Fprintf(&b, "| 结构截断 Episode | %s |\n", note)
	} else {
		fmt.Fprintf(&b, "| 结构截断 Episode | 关闭（仅至反向信号） |\n")
	}
	if flow.DiagnoseVerdict != "" {
		fmt.Fprintf(&b, "| 诊断结论 | %s |\n", flow.DiagnoseVerdict)
	}
	if flow.DiagnoseSummary != "" {
		fmt.Fprintf(&b, "\n**diagnose 摘要**：%s\n", flow.DiagnoseSummary)
	}
	return b.String()
}

func appendSignalEvalMethodology(b *strings.Builder) {
	fmt.Fprintf(b, "\n### 命中率与有效性怎么算\n\n")
	fmt.Fprintf(b, "1. **Strict（闭环）**：买段 → 下一**卖**前一根 K；卖段 → 下一**买**前一根。仅闭合段计入 strict 命中率。\n")
	fmt.Fprintf(b, "2. **Path（诊断）**：若无反向信号，则截到**下一同向买/卖**前一根；最后一段截到样本末（`open_tail`）。\n")
	fmt.Fprintf(b, "3. **Path 指标**：段内最高价涨幅 `peak_return`、自入场/自高点 `max_drawdown`（用 high/low）。\n")
	fmt.Fprintf(b, "4. **命中**：买段方向收益 > 0；卖段 < 0（卖段方向收益取反便于阅读）。\n")
	fmt.Fprintf(b, "5. **仅买无卖**时请看 **Path 汇总**，strict 命中率可能为 N/A。\n")
	fmt.Fprintf(b, "6. **结构截断（可选）**：买/卖 Strict 段按配置的参考线（默认买 `support_high`、卖 `resist_low`）逐 K 与 **as-of 该 bar 的 Key Level 序列**比较提前结束（probe `key_levels_series`，默认按日对齐重算，无未来函数）。\n")
}

func appendSignalEvalSection(b *strings.Builder, eval slots.SignalEpisodeEval) {
	fmt.Fprintf(b, "\n### 汇总 · Strict（%s · 至反向信号）\n\n", eval.Method)
	fmt.Fprintf(b, "| 方向 | 闭合段 | 未闭合 | 命中/闭合 | 命中率 | 均方向收益 | 中位方向收益 | 均持有K线 | 中位持有K线 |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	writeEvalRow(b, "买入", eval.BuyEpisodes)
	writeEvalRow(b, "卖出", eval.SellEpisodes)
	fmt.Fprintf(b, "\n### 汇总 · Path（诊断：至下一同向或样本末）\n\n")
	fmt.Fprintf(b, "| 方向 | 段数 | 命中/段 | 命中率 | 均方向收益 | 均 peak | 均 maxDD(自高点) | 均持有K线 |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	writePathEvalRow(b, "买入", eval.BuyPathEpisodes)
	writePathEvalRow(b, "卖出", eval.SellPathEpisodes)
}

func writeEvalRow(b *strings.Builder, label string, m slots.SignalEpisodeMetrics) {
	hit := "-"
	ratio := "-"
	avg := "-"
	med := "-"
	avgHold := "-"
	medHold := "-"
	if m.CompleteCount > 0 {
		hit = fmt.Sprintf("%d/%d", m.HitCount, m.CompleteCount)
		ratio = fmt.Sprintf("%.0f%%", m.HitRate*100)
		avg = slots.FormatPct(m.AvgReturn)
		med = slots.FormatPct(m.MedianReturn)
		avgHold = fmt.Sprintf("%.1f", m.AvgHoldingBars)
		medHold = fmt.Sprintf("%.0f", m.MedianHoldingBars)
	}
	fmt.Fprintf(b, "| %s | %d | %d | %s | %s | %s | %s | %s | %s |\n",
		label, m.CompleteCount, m.IncompleteCount, hit, ratio, avg, med, avgHold, medHold)
}

func appendEpisodeDetailSections(b *strings.Builder, eval slots.SignalEpisodeEval) {
	appendEpisodeDetailTable(b, "买入 Episode 明细", eval.BuyDetails)
	appendEpisodeDetailTable(b, "卖出 Episode 明细", eval.SellDetails)
}

func writePathEvalRow(b *strings.Builder, label string, m slots.SignalEpisodeMetrics) {
	hit := "-"
	ratio := "-"
	avg := "-"
	peak := "-"
	dd := "-"
	avgHold := "-"
	if m.CompleteCount > 0 {
		hit = fmt.Sprintf("%d/%d", m.HitCount, m.CompleteCount)
		ratio = fmt.Sprintf("%.0f%%", m.HitRate*100)
		avg = slots.FormatPct(m.AvgReturn)
		peak = slots.FormatPct(m.AvgPeakReturn)
		dd = slots.FormatPct(m.AvgMaxDrawdown)
		avgHold = fmt.Sprintf("%.1f", m.AvgHoldingBars)
	}
	fmt.Fprintf(b, "| %s | %d | %s | %s | %s | %s | %s | %s |\n",
		label, m.CompleteCount, hit, ratio, avg, peak, dd, avgHold)
}

func appendEpisodeDetailTable(b *strings.Builder, title string, details []slots.SignalEpisodeDetail) {
	fmt.Fprintf(b, "\n### %s\n\n", title)
	if len(details) == 0 {
		fmt.Fprintf(b, "_无 episode_\n")
		return
	}
	fmt.Fprintf(b, "| # | 信号时间 | 起点价 | Strict | Path 终点 | peak | maxDD | Path 方向收益 | Path 命中 |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for i, ep := range details {
		strict := "未闭合"
		if ep.Complete {
			strict = fmt.Sprintf("%s · %s", dashTime(ep.EndTime), slots.FormatPct(ep.DirectionReturn))
		}
		pathEnd := "—"
		pathRet := "—"
		pathHit := "—"
		peak := "—"
		dd := "—"
		if ep.PathEvaluable {
			pathEnd = fmt.Sprintf("%s (%s) · %s", dashTime(ep.PathEndTime), ep.PathEndReason, slots.FormatPct(ep.PathDirectionReturn))
			pathRet = slots.FormatPct(ep.PathDirectionReturn)
			if ep.PathHit {
				pathHit = "是"
			} else {
				pathHit = "否"
			}
			peak = slots.FormatPct(ep.PeakReturn)
			dd = slots.FormatPct(ep.MaxDrawdownFromPeak)
		}
		fmt.Fprintf(b, "| %d | %s | %.2f | %s | %s | %s | %s | %s | %s |\n",
			i+1, dashTime(ep.StartTime), ep.StartClose, strict, pathEnd, peak, dd, pathRet, pathHit)
	}
}

func dashTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" {
		return "—"
	}
	return s
}
