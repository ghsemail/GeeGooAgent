package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

// ShouldSkipRetrievalGate reports whether the turn can skip long-term recall.
// Only clarify turns skip: the option sheet must open before extra FTS work.
func ShouldSkipRetrievalGate(matchedSkills []string, plan cognition.TurnPlan, userText string) bool {
	_ = matchedSkills
	_ = userText
	return plan.Mode == cognition.ModeClarify
}

func skipRetrievalReason(matchedSkills []string, plan cognition.TurnPlan, userText string) string {
	_ = matchedSkills
	_ = userText
	if plan.Mode == cognition.ModeClarify {
		return "clarify turn"
	}
	return "skip"
}
