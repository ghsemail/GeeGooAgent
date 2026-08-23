package agent_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
)

func TestShouldSkipRetrievalGate(t *testing.T) {
	tests := []struct {
		skills []string
		want   bool
	}{
		{nil, false},
		{[]string{"knowledge-base"}, false},
		{[]string{"strategy-backtest-history"}, true},
		{[]string{"strategy-signal-probe", "strategy-backtest"}, true},
		{[]string{"knowledge-base", "strategy-backtest-run"}, true},
	}
	for _, tc := range tests {
		got := agent.ShouldSkipRetrievalGate(tc.skills)
		if got != tc.want {
			t.Fatalf("skills=%v got=%v want=%v", tc.skills, got, tc.want)
		}
	}
}
