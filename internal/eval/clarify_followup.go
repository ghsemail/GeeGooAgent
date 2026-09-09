package eval

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// DialogueExecutionPlan splits scripted user turns into regular vs on-clarify follow-ups.
func DialogueExecutionPlan(opts TurnPlanCaseOptions) (regular, clarify []EvalDialogueTurn) {
	opts = opts.Normalize()
	for _, turn := range opts.Dialogue {
		if turn.Role != "user" {
			continue
		}
		text := strings.TrimSpace(turn.Text)
		if text == "" {
			continue
		}
		t := EvalDialogueTurn{Role: "user", Text: text, OnClarify: turn.OnClarify, Judge: turn.Judge}
		if turn.OnClarify {
			clarify = append(clarify, t)
			continue
		}
		regular = append(regular, t)
	}
	return regular, clarify
}

// ClarifyDefaultTexts returns configured default user replies for clarify tool callbacks.
func ClarifyDefaultTexts(opts TurnPlanCaseOptions) []string {
	opts = opts.Normalize()
	seen := map[string]struct{}{}
	out := []string{}
	for _, turn := range opts.Dialogue {
		if !turn.OnClarify {
			continue
		}
		text := strings.TrimSpace(turn.Text)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	if reply := strings.TrimSpace(opts.ClarifyReply); reply != "" {
		if _, ok := seen[reply]; !ok {
			out = append(out, reply)
		}
	}
	return out
}

// NeedsClarifyFollowup reports whether scripted on-clarify turns should run after the primary dialogue.
func NeedsClarifyFollowup(chat *chatsession.ChatSession, opts TurnPlanCaseOptions) bool {
	_, clarify := DialogueExecutionPlan(opts)
	if len(clarify) == 0 {
		return false
	}
	spec := opts.Normalize().execution()
	if len(spec.LegacyRequireTools) == 0 {
		return true
	}
	trace := chatsession.TurnToolsTraceFromSession(chat)
	sessionTools := chatsession.SessionToolsFromTrace(trace)
	for _, tool := range spec.LegacyRequireTools {
		if !containsString(sessionTools, tool) {
			return true
		}
	}
	return false
}

// PickClarifyAnswer matches case-configured defaults against clarify choices or returns the first default.
func PickClarifyAnswer(question string, choices []string, defaults []string) (string, bool) {
	defaults = nonEmptyStrings(defaults)
	if len(defaults) == 0 {
		return "", false
	}
	if len(choices) > 0 {
		for _, def := range defaults {
			def = strings.TrimSpace(def)
			if def == "" {
				continue
			}
			for _, choice := range choices {
				if clarifyChoiceMatches(def, choice) {
					return choice, true
				}
			}
		}
		return "", false
	}
	return defaults[0], true
}

func clarifyChoiceMatches(def, choice string) bool {
	def = strings.ToLower(strings.TrimSpace(def))
	choice = strings.ToLower(strings.TrimSpace(choice))
	if def == "" || choice == "" {
		return false
	}
	if strings.Contains(choice, def) || strings.Contains(def, choice) {
		return true
	}
	keywords := []string{"sar", "macd", "rsi", "ema", "dca", "grid", "吊灯", "共振"}
	needed := 0
	matched := 0
	for _, kw := range keywords {
		if strings.Contains(def, kw) {
			needed++
			if strings.Contains(choice, kw) {
				matched++
			}
		}
	}
	if needed == 0 {
		return false
	}
	return matched == needed
}

func nonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
