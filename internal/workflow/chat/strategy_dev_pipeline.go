package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const PhaseDevPipelineRun = "dev_pipeline_run"

var freqInTextPattern = regexp.MustCompile(`(?i)(5|15|60)\s*分钟|(?i)(5m|15m|60m|daily|weekly|1wk|日线|周线)`)

func timeNowUTC() time.Time { return time.Now().UTC() }

func newStrategyDevPipelineFlow(userText string, opts *runtime.StrategyDevOpts) *Flow {
	now := timeNowUTC()
	code, freq, months := parseStrategyDevPipelineInput(userText, opts)
	return &Flow{
		RunID:        newRunID(),
		Template:     SkillStrategyDevPipeline,
		Status:       StatusRunning,
		Phase:        PhaseDevPipelineRun,
		WorkflowStep: StepStrategyDevelop,
		TriggerText:  strings.TrimSpace(userText),
		StockQuery:   code,
		StockCode:    code,
		ProbeFrequencyOverride: freq,
		MonthsBack:   months,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func parseStrategyDevPipelineInput(text string, opts *runtime.StrategyDevOpts) (code, freq string, months int) {
	if opts != nil {
		if c := strings.TrimSpace(opts.Code); c != "" {
			code = c
		}
		if f := strings.TrimSpace(opts.Frequency); f != "" {
			freq = strategydev.NormalizeFrequency(f)
		}
		if opts.MonthsBack > 0 {
			months = opts.MonthsBack
		}
	}
	if code == "" {
		code = slots.ExtractExplicitStockReference(text)
	}
	if freq == "" {
		if m := freqInTextPattern.FindString(text); m != "" {
			freq = strategydev.NormalizeFrequency(m)
		}
	}
	if months <= 0 {
		months = defaultMonthsBack
	}
	return code, freq, strategydev.CapMonthsBack(months)
}

func (r *Runner) advanceStrategyDevPipeline(
	ctx context.Context,
	session *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	if flow.Phase != PhaseDevPipelineRun {
		flow.Phase = PhaseDevPipelineRun
	}
	code := strings.TrimSpace(flow.StockCode)
	if code == "" && session != nil {
		code = strings.TrimSpace(flow.StockQuery)
	}
	freq := strings.TrimSpace(flow.ProbeFrequencyOverride)
	if code == "" || freq == "" {
		return terminalError("策略开发编排需要标的与周期，例如：策略开发 00700.HK 60分钟")
	}
	jev := strategydev.NewJevClient(strategydev.JevConfig{})
	if r != nil && r.JevConfig != nil {
		jev = strategydev.NewJevClient(*r.JevConfig)
	}
	runTool := func(c context.Context, name string, args map[string]any) tools.Result {
		return r.runTool(c, toolCtx, name, args, recordTool)
	}
	res, err := strategydev.Run(ctx, strategydev.Input{
		Code: code, Frequency: freq, MonthsBack: flow.MonthsBack,
	}, strategydev.Deps{Run: runTool, Jev: jev})
	if err != nil {
		return err
	}
	flow.StrategyDevPipelineResult = &res
	flow.Phase = PhaseSummarize
	flow.Status = StatusCompleted
	flow.touch()
	return nil
}

func renderStrategyDevPipelineReport(flow *Flow) string {
	if flow.StrategyDevPipelineResult == nil {
		return "策略开发编排未完成"
	}
	r := flow.StrategyDevPipelineResult
	var b strings.Builder
	fmt.Fprintf(&b, "## 策略开发编排 · %s · %s\n\n", r.Code, r.Frequency)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 1 取数据 | %d 根 K 线（limit=%d） |\n", r.BarLimit, r.BarLimit)
	fmt.Fprintf(&b, "| 2 认行情 | %s · %s", r.Regime, r.Side)
	if r.RegimeNote != "" {
		fmt.Fprintf(&b, "（%s）", r.RegimeNote)
	}
	fmt.Fprintf(&b, " |\n")
	if r.Skipped {
		fmt.Fprintf(&b, "| 3–6 | 跳过：%s |\n", r.SkipReason)
		return strings.TrimSpace(b.String())
	}
	if r.SelectedBuy != nil {
		fmt.Fprintf(&b, "| 4 选中买入 | %s (%s) |\n", r.SelectedBuy.Name, r.SelectedBuy.SignalID)
	}
	if r.SelectedSell != nil {
		fmt.Fprintf(&b, "| 4 选中卖出 | %s (%s) |\n", r.SelectedSell.Name, r.SelectedSell.SignalID)
	}
	if r.BuyFallbackNote != "" {
		fmt.Fprintf(&b, "| 平空买入 | %s |\n", r.BuyFallbackNote)
	}
	fmt.Fprintf(&b, "| 5 诊断 | 买 %d / 卖 %d · 完成 %d 段 · 命中率 %.0f%% · 平均方向收益 %.2f%% · 累计 %.2f%% |\n",
		r.Diagnose.BuyHits, r.Diagnose.SellHits, r.Diagnose.CompleteSegs,
		r.Diagnose.HitRate*100, r.Diagnose.AvgDirection*100, r.Diagnose.CumPoints)
	fmt.Fprintf(&b, "| 6 结论 | **%s**", r.FinalVerdict)
	if r.VerdictReason != "" {
		fmt.Fprintf(&b, " — %s", r.VerdictReason)
	}
	fmt.Fprintf(&b, " |\n")
	return strings.TrimSpace(b.String())
}

func renderStrategyDevPipelinePartial(flow *Flow) string {
	return fmt.Sprintf("## 策略开发编排\n\n- 标的：%s\n- 周期：%s\n- 阶段：运行六步 workflow",
		strings.TrimSpace(flow.StockCode), strings.TrimSpace(flow.ProbeFrequencyOverride))
}
