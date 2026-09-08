package eval

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

// DefaultTurnPlanPlanner returns an IntentPlanner backed by a classify fixture
// that mirrors DefaultTurnPlanSuite expectations (for offline regression).
func DefaultTurnPlanPlanner() cognition.Planner {
	return cognition.IntentPlanner{LLM: TurnPlanClassifyFixture(DefaultTurnPlanSuite())}
}

// TurnPlanClassifyFixture builds a classify LLM fixture from suite turn expectations.
func TurnPlanClassifyFixture(suite TurnPlanSuite) *cognition.ClassifyFixtureProvider {
	byMessage := make(map[string]string, len(suite.Turns))
	for _, turn := range suite.Turns {
		msg := strings.TrimSpace(turn.Message)
		if msg == "" {
			continue
		}
		byMessage[msg] = cognition.FormatClassifyJSON(
			turn.ExpectDomain,
			turn.ExpectMode,
			turn.ExpectAct,
			"turnplan fixture",
		)
	}
	return &cognition.ClassifyFixtureProvider{ByMessage: byMessage}
}
