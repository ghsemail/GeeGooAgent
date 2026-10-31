package chat

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// BenchmarkComposeFlow is a representative flow for compose latency benchmarks (AAPL-like).
func BenchmarkComposeFlow() *Flow {
	return &Flow{
		Template:       SkillSignalDiagnose,
		CatalogType:    "combination",
		CatalogLabel:   "SAR MACD",
		StockName:      "Apple",
		StockCode:      "AAPL.US",
		MonthsBack:     3,
		ProbeBuyHits:   5,
		ProbeSellHits:  4,
		ProbeBarCount:  120,
		DiagnoseVerdict: "可用",
		DiagnoseSummary: "probe 与 episode 指标整体一致",
		Strategies:     []StrategyItem{{Frequency: "60m", Label: "SAR MACD"}},
	}
}

// BuildSignalDiagnoseComposeMessages builds the same messages as diag_compose LLM call.
func BuildSignalDiagnoseComposeMessages(flow *Flow) []llm.Message {
	if flow == nil {
		flow = BenchmarkComposeFlow()
	}
	label := strategyDisplayLabel(flow)
	freq := "60m"
	if len(flow.Strategies) > 0 && flow.Strategies[0].Frequency != "" {
		freq = flow.Strategies[0].Frequency
	}
	user := buildSignalDiagnoseComposeUserPrompt(flow, label, freq)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: signalDiagnoseJudgmentSystemPrompt()},
		{Role: llm.RoleUser, Content: user},
	}
}

func buildSignalDiagnoseComposeUserPrompt(flow *Flow, label, freq string) string {
	longOnlyNote := ""
	if flow.TradeDirection.LongOnlyEpisodeEval() {
		longOnlyNote = "\ntrade_direction=long_only（只做多；probe 卖次为平多卖出信号，勿表述为做空 Episode 或「卖出触发 N 次」暗示开空）"
	}
	return fmt.Sprintf(`请用完整、正式的中文写信号诊断评价（可分段落，勿编造数字；不要输出英文推理或草稿，只输出最终中文正文）：

策略 %s（%s）· %s %s · %s / %d月
probe %s（%d根K）
diagnose：%s — %s
Episode：%s%s`,
		label, flow.CatalogType,
		flow.StockName, flow.StockCode,
		freq, flow.MonthsBack,
		strategydev.SignalDiagnoseProbeHitLine(flow.TradeDirection.LongOnlyEpisodeEval(), flow.ProbeBuyHits, flow.ProbeSellHits),
		flow.ProbeBarCount,
		flow.DiagnoseVerdict, flow.DiagnoseSummary,
		slots.FormatEvalMetricsJSON(flow.SignalEval),
		longOnlyNote,
	)
}

// extract from synthesize - refactor signal_diagnose_compose to call buildSignalDiagnoseComposeUserPrompt
