package eval

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// JudgedUserTurnIndex returns the 1-based user-turn index marked judge in the dialogue script.
func JudgedUserTurnIndex(opts TurnPlanCaseOptions, chat *chatsession.ChatSession) int {
	opts = opts.Normalize()
	regular, clarify := DialogueExecutionPlan(opts)
	idx := 0
	for _, turn := range regular {
		idx++
		if turn.Judge {
			return idx
		}
	}
	if len(clarify) > 0 && NeedsClarifyFollowup(chat, opts) {
		for _, turn := range clarify {
			idx++
			if turn.Judge {
				return idx
			}
		}
	}
	if idx == 0 {
		return 1
	}
	return idx
}

// HasJudgeTurn reports whether any scripted user turn is marked judge.
func HasJudgeTurn(opts TurnPlanCaseOptions) bool {
	opts = opts.Normalize()
	for _, turn := range opts.Dialogue {
		if turn.Role != "user" {
			continue
		}
		if strings.TrimSpace(turn.Text) == "" {
			continue
		}
		if turn.Judge {
			return true
		}
	}
	return false
}
