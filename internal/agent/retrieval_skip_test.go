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
		want   bool
	}{
		{nil, cognition.TurnPlan{}, false},
		{[]string{"knowledge-base"}, cognition.TurnPlan{}, false},
		{[]string{"strategy-backtest-history"}, cognition.TurnPlan{}, true},
		{[]string{"strategy-signal-probe", "strategy-backtest"}, cognition.TurnPlan{}, true},
		{[]string{"knowledge-base", "strategy-backtest-run"}, cognition.TurnPlan{}, true},
		{nil, clarifyPlan, true},
		{[]string{"knowledge-base"}, clarifyPlan, true},
	}
	for _, tc := range tests {
		got := agent.ShouldSkipRetrievalGate(tc.skills, tc.plan)
		if got != tc.want {
			t.Fatalf("skills=%v plan=%s got=%v want=%v", tc.skills, tc.plan.Mode, got, tc.want)
		}
	}
}
