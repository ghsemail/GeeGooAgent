package decision

import (
	"context"
	"strings"
)

const defaultMinConfidence = 0.85

// Coordinator applies Mode policy: shadow/paper never take over; failures fall back to LLM.
type Coordinator struct {
	Mode          Mode
	Hook          Hook
	MinConfidence float64
}

// Consult runs the hook when enabled and emits a compact `decision` SSE event.
// The second return is true only when ModeReplace allows takeover (never for plan_gate).
func (c *Coordinator) Consult(ctx context.Context, q Question, emit ProgressFunc) (Answer, bool) {
	if c == nil || c.Mode == ModeOff || c.Hook == nil {
		return Answer{QuestionID: q.ID, Who: WhoLLM}, false
	}
	ans, err := c.Hook.Decide(ctx, q)
	if err != nil {
		ans = Answer{
			QuestionID: q.ID,
			Who:        WhoFallback,
			Reason:     err.Error(),
		}
	} else if ans.Who == "" {
		if err == nil {
			ans.Who = WhoJev
		}
	}
	if strings.TrimSpace(ans.QuestionID) == "" {
		ans.QuestionID = q.ID
	}

	takeover := c.takeoverAllowed(q, ans, err)
	if emit != nil {
		emit("decision", map[string]any{
			"question_id": ans.QuestionID,
			"kind":        string(q.Kind),
			"who":         string(ans.Who),
			"answer":      ans.ChoiceID,
			"confidence":  ans.Confidence,
			"mode":        string(c.Mode),
			"takeover":    takeover,
		})
	}
	return ans, takeover
}

func (c *Coordinator) takeoverAllowed(q Question, ans Answer, hookErr error) bool {
	if c.Mode != ModeReplace || hookErr != nil {
		return false
	}
	min := c.MinConfidence
	if min <= 0 {
		min = defaultMinConfidence
	}
	if ans.Confidence < min || strings.TrimSpace(ans.ChoiceID) == "" {
		return false
	}
	// Plan Gate and clarify must stay on main LLM + existing approval UX until explicitly designed.
	if q.Kind == KindPlanGate || q.Kind == KindClarify {
		return false
	}
	return q.Kind == KindEndTurn || q.Kind == KindToolPlan
}
