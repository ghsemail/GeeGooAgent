package agent

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

// PlanStepsMarkdown lists plan steps per domain for dashboard reference.
func PlanStepsMarkdown() string {
	cases := []struct {
		label string
		plan  cognition.TurnPlan
	}{
		{"signal_probe / execute", cognition.TurnPlan{Domain: cognition.DomainSignalProbe, Mode: cognition.ModeExecute}},
		{"backtest_run / execute", cognition.TurnPlan{Domain: cognition.DomainBacktestRun, Mode: cognition.ModeExecute}},
		{"backtest_history / gather", cognition.TurnPlan{Domain: cognition.DomainBacktestHistory, Mode: cognition.ModeGather}},
		{"stock_analysis / quote_price", cognition.TurnPlan{
			Domain: cognition.DomainStockAnalysis,
			Mode:   cognition.ModeGather,
			Act:    string(domaincatalog.StockActQuotePrice),
		}},
		{"stock_analysis / multi_symbol", cognition.TurnPlan{
			Domain: cognition.DomainStockAnalysis,
			Mode:   cognition.ModeGather,
			Act:    string(domaincatalog.StockActMultiSymbol),
		}},
		{"dca_grid / gather", cognition.TurnPlan{Domain: cognition.DomainDCAGrid, Mode: cognition.ModeGather}},
		{"dca_grid / execute (生成方案)", cognition.TurnPlan{Domain: cognition.DomainDCAGrid, Mode: cognition.ModeExecute}},
		{"bot_manage / gather", cognition.TurnPlan{Domain: cognition.DomainBotManage, Mode: cognition.ModeGather}},
		{"ambiguous / clarify", cognition.TurnPlan{Domain: cognition.DomainAmbiguous, Mode: cognition.ModeClarify}},
	}
	var b strings.Builder
	b.WriteString("## Plan Steps（按 domain）\n\n")
	b.WriteString("由 classify 结果生成，注入 ReAct 的 `plan steps` 与 SSE `turn_plan.plan_steps` 同源。\n\n")
	for _, c := range cases {
		steps := cursorPlanSteps(c.plan)
		if len(steps) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n", c.label)
		for i, step := range steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, step)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TurnPlanFragmentTemplateDoc describes the turn plan system fragment shape.
func TurnPlanFragmentTemplateDoc() string {
	return `## Plan Fragment 模板（agent_context）

每轮 classify 后注入 system fragment（软约束：推荐步骤与工具，**不裁** tool schema）：

` + "```text\n" + strings.TrimSpace(`Turn plan (soft guidance — follow numbered steps and preferred tools when they fit; full tool schema stays available):
- domain: …
- act: …
- mode: …
- reason: …
- execution profile: …
- execution contract: …
- plan steps:
  1. …
  2. …
- skills: …
- preferred tools: …
- execute the plan above; deviate only when session context or user text clearly requires it`) + "\n```\n"
}
