package agent

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func TestCursorTurnPlanFragmentIncludesStepsAndProfile(t *testing.T) {
	plan := cognition.TurnPlan{
		Domain: cognition.DomainSignalProbe,
		Act:    "probe",
		Mode:   cognition.ModeExecute,
		Reason: "llm: probe intent",
		Skills: []string{"strategy-signal-probe"},
		ToolsAllow: []string{
			"clarify", "probe_bot_signal_series", "resolve_signal",
		},
	}
	text := turnPlanFragment(plan, "测一下腾讯 SAR+MACD 买卖点", true).Render()
	if !strings.Contains(text, "soft guidance") {
		t.Fatalf("expected soft guidance header, got %q", text)
	}
	if !strings.Contains(text, "plan steps:") {
		t.Fatalf("expected plan steps, got %q", text)
	}
	if !strings.Contains(text, "preferred tools:") {
		t.Fatalf("expected preferred tools, got %q", text)
	}
	if !strings.Contains(text, "execution profile:") {
		t.Fatalf("expected execution profile, got %q", text)
	}
}

func TestCursorPlanStepsBacktest(t *testing.T) {
	steps := cursorPlanSteps(cognition.TurnPlan{
		Domain: cognition.DomainBacktestRun,
		Mode:   cognition.ModeExecute,
	})
	if len(steps) < 3 || !strings.Contains(steps[0], "strategy-backtest-run") {
		t.Fatalf("steps=%v", steps)
	}
	if !strings.Contains(steps[2], "run_strategy_backtest") {
		t.Fatalf("steps=%v", steps)
	}
}
