package diagnoseskill

import (
	"context"
	"testing"
	"time"
)

// Each layer's worst case has to fit the 15 minute call and BFF timeout.
func TestMacroNewsTimeBudgetsFitTheCallTimeout(t *testing.T) {
	if macroNewsCallTimeout != 15*time.Minute {
		t.Fatalf("callers wait 15 minutes, not %v", macroNewsCallTimeout)
	}
	for name, worst := range map[string]time.Duration{
		"items only":         macroNewsItemBudget + macroNewsModelCallMax,
		"pool rollup only":   macroNewsPoolBudget,
		"items then rollup":  macroNewsItemWithRollupBudget + macroNewsModelCallMax + macroNewsPoolBudget,
		"cross-pool summary": macroNewsOverviewBudget,
	} {
		// A minute is left for the network and the reply.
		if worst > macroNewsCallTimeout-time.Minute {
			t.Errorf("%s takes up to %v", name, worst)
		}
	}
	if macroNewsPoolBudget < macroNewsDecisionCallMax+macroNewsModelCallMax {
		t.Error("the pool budget must hold one Choice and one summary call")
	}
}

type neverCalledLLM struct{ t *testing.T }

func (n neverCalledLLM) Complete(_ context.Context, _, _ string, _ func(string) error) (string, error) {
	n.t.Error("no model call may start after the budget")
	return "", nil
}

func TestMacroNewsItemsBeyondTheBudgetAreLeftOver(t *testing.T) {
	items := []MacroNewsItem{{NewsID: "a", Title: "t"}, {NewsID: "b", Title: "t"}}
	out, _ := analyzeMacroNewsItems(context.Background(), neverCalledLLM{t}, "hk", "港股池", items, -time.Second)
	for _, item := range out {
		if item.AnalysisStatus != itemStatusFailed || item.ErrorCode != itemErrorBudget || item.JEVDirection != "" {
			t.Fatalf("item=%+v", item)
		}
	}
}
