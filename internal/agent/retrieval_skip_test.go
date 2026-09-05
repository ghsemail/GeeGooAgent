package agent_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func TestShouldSkipRetrievalGate(t *testing.T) {
	clarifyPlan := cognition.TurnPlan{Mode: cognition.ModeClarify}
	tests := []struct {
		skills []string
		plan   cognition.TurnPlan
		text   string
		want   bool
	}{
		{nil, cognition.TurnPlan{}, "还记得上次的标的吗", false},
		{[]string{"knowledge-base"}, cognition.TurnPlan{}, "还记得上次", false},
		{[]string{"knowledge-base"}, cognition.TurnPlan{}, "MACD", true},
		{[]string{"strategy-backtest-history"}, cognition.TurnPlan{}, "还记得上次", true},
		{[]string{"strategy-signal-probe", "strategy-backtest"}, cognition.TurnPlan{}, "帮我测信号", true},
		{[]string{"knowledge-base", "strategy-backtest-run"}, cognition.TurnPlan{}, "帮我回测", true},
		{nil, clarifyPlan, "MACD", true},
		{[]string{"knowledge-base"}, clarifyPlan, "还记得", true},
	}
	for _, tc := range tests {
		got := agent.ShouldSkipRetrievalGate(tc.skills, tc.plan, tc.text)
		if got != tc.want {
			t.Fatalf("skills=%v plan=%s text=%q got=%v want=%v", tc.skills, tc.plan.Mode, tc.text, got, tc.want)
		}
	}
}
