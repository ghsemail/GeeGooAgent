package chat

import (
	"context"
	"fmt"
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
	flow.Phase = PhaseDiagSaveKB
	flow.touch()
	return nil
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

Episode 评价（至下一次反向信号，同向连续触发合并；未闭合段不计入主命中率）：
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
	if buy.CompleteCount+sell.CompleteCount == 0 {
		b.WriteString("可闭合样本不足，建议延长回溯或换周期后再评。")
		return strings.TrimSpace(b.String())
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
		fmt.Fprintf(&b, " 另有 %d 段未闭合。", buy.IncompleteCount+sell.IncompleteCount)
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
	appendSignalEvalSection(&b, flow.SignalEval)
	fmt.Fprintf(&b, "\n### Agent 评价\n\n%s\n", strings.TrimSpace(judgment))
	appendDiagnoseRuleDetails(&b, flow.DiagnoseRaw)
	fmt.Fprintf(&b, "\n> Workflow：read_strategy → probe → evaluate_accuracy → diagnose → LLM 评价 → 写入知识库（%s）。\n",
		tools.SignalDiagnoseFolder)
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
	fmt.Fprintf(&b, "| 信号测试 probe | 买 %d 次 / 卖 %d 次", flow.ProbeBuyHits, flow.ProbeSellHits)
	if flow.ProbeBarCount > 0 {
		fmt.Fprintf(&b, "（%d 根 K 线）", flow.ProbeBarCount)
	}
	fmt.Fprintf(&b, " |\n")
	if flow.DiagnoseVerdict != "" {
		fmt.Fprintf(&b, "| 诊断结论 | %s |\n", flow.DiagnoseVerdict)
	}
	if flow.DiagnoseSummary != "" {
		fmt.Fprintf(&b, "\n**diagnose 摘要**：%s\n", flow.DiagnoseSummary)
	}
	return b.String()
}

func appendSignalEvalSection(b *strings.Builder, eval slots.SignalEpisodeEval) {
	fmt.Fprintf(b, "\n### 信号准确率（Episode · %s）\n\n", eval.Method)
	fmt.Fprintf(b, "| 方向 | 闭合段 | 未闭合 | 命中率 | 均收益 | 中位收益 |\n| --- | --- | --- | --- | --- | --- |\n")
	writeEvalRow(b, "买入", eval.BuyEpisodes)
	writeEvalRow(b, "卖出", eval.SellEpisodes)
	fmt.Fprintf(b, "\n> 口径：起点收盘价 → 下一次反向信号前一根收盘价；同向连续触发合并为一段。\n")
}

func writeEvalRow(b *strings.Builder, label string, m slots.SignalEpisodeMetrics) {
	hit := "-"
	avg := "-"
	med := "-"
	if m.CompleteCount > 0 {
		hit = fmt.Sprintf("%.0f%%", m.HitRate*100)
		avg = slots.FormatPct(m.AvgReturn)
		med = slots.FormatPct(m.MedianReturn)
	}
	fmt.Fprintf(b, "| %s | %d | %d | %s | %s | %s |\n", label, m.CompleteCount, m.IncompleteCount, hit, avg, med)
}
