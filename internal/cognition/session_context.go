package cognition

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const (
	planContextMaxTurns     = 8
	planContextMaxUserRunes = 400
	planContextMaxAsstRunes = 600
	planContextMaxToolRunes = 220
	planContextMaxSummary   = 1200
	planContextMaxTotal     = 4500
)

// PlanSessionView is the session slice passed into intent classification.
type PlanSessionView struct {
	Ctx             context.Context
	Messages        []llm.Message
	UserText        string
	LastDomain      Domain
	PreviousSummary string
}

// PlanInputWithUserDialogue builds PlanInput for plan-only eval from prior user turns.
func PlanInputWithUserDialogue(ctx context.Context, userText string, lastDomain Domain, priorUserTurns []string) PlanInput {
	prior := append([]string(nil), priorUserTurns...)
	return PlanInput{
		Ctx:              ctx,
		UserText:         userText,
		LastDomain:       lastDomain,
		RecentDialogue:   FormatUserTurns(prior),
		sessionHistory:   len(prior) > 0,
	}
}

// BuildPlanInput builds a PlanInput with compressed session continuity for classify.
func BuildPlanInput(v PlanSessionView) PlanInput {
	recent, hasHistory := FormatRecentDialogue(v.Messages, v.UserText)
	return PlanInput{
		Ctx:              v.Ctx,
		UserText:         v.UserText,
		LastDomain:       v.LastDomain,
		SessionSummary:   strings.TrimSpace(v.PreviousSummary),
		RecentDialogue:   recent,
		sessionHistory:   hasHistory,
	}
}

// HasSessionHistory reports whether classify should treat this as a continuing session.
func (in PlanInput) HasSessionHistory() bool {
	if in.sessionHistory {
		return true
	}
	return strings.TrimSpace(in.RecentDialogue) != "" || strings.TrimSpace(in.SessionSummary) != ""
}

// FormatRecentDialogue renders prior session turns for the classify prompt.
// The current user utterance is omitted — it is appended separately as "User:".
func FormatRecentDialogue(msgs []llm.Message, currentUser string) (formatted string, hasHistory bool) {
	currentUser = strings.TrimSpace(currentUser)
	end := len(msgs)
	if end > 0 {
		last := msgs[end-1]
		if last.Role == llm.RoleUser && strings.TrimSpace(last.Content) == currentUser {
			end--
		}
	}

	type line struct {
		text string
	}
	lines := make([]line, 0, planContextMaxTurns)
	total := 0

	for i := end - 1; i >= 0 && len(lines) < planContextMaxTurns; i-- {
		m := msgs[i]
		if m.Role == llm.RoleSystem {
			continue
		}
		formattedLine, ok := formatPlanContextMessage(m)
		if !ok {
			continue
		}
		if total+len([]rune(formattedLine)) > planContextMaxTotal {
			break
		}
		lines = append(lines, line{text: formattedLine})
		total += len([]rune(formattedLine))
		hasHistory = true
	}

	if !hasHistory {
		return "", false
	}
	out := make([]string, len(lines))
	for i := range lines {
		out[len(lines)-1-i] = lines[i].text
	}
	return strings.Join(out, "\n"), true
}

// FormatUserTurns formats prior user-only turns (plan-only eval / suggest helpers).
func FormatUserTurns(turns []string) string {
	if len(turns) == 0 {
		return ""
	}
	var b strings.Builder
	for _, t := range turns {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("user: ")
		b.WriteString(truncateRunes(t, planContextMaxUserRunes))
	}
	return b.String()
}

func formatPlanContextMessage(m llm.Message) (string, bool) {
	switch m.Role {
	case llm.RoleUser:
		text := strings.TrimSpace(m.Content)
		if text == "" {
			return "", false
		}
		return "user: " + truncateRunes(text, planContextMaxUserRunes), true
	case llm.RoleAssistant:
		text := strings.TrimSpace(m.Content)
		if text == "" && len(m.ToolCalls) == 0 {
			return "", false
		}
		prefix := "assistant"
		if names := toolCallNames(m.ToolCalls); names != "" {
			prefix += " [" + names + "]"
		}
		if text == "" {
			return prefix + ": (tool calls)", true
		}
		return prefix + ": " + truncateRunes(text, planContextMaxAsstRunes), true
	case llm.RoleTool:
		text := strings.TrimSpace(m.Content)
		if text == "" {
			return "", false
		}
		label := "tool"
		if id := strings.TrimSpace(m.ToolCallID); id != "" {
			label = "tool(" + truncateRunes(id, 40) + ")"
		}
		return label + ": " + truncateRunes(text, planContextMaxToolRunes), true
	default:
		return "", false
	}
}

func toolCallNames(calls []llm.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	names := make([]string, 0, len(calls))
	seen := map[string]struct{}{}
	for _, c := range calls {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func buildClassifyPrompt(in PlanInput) string {
	var b strings.Builder
	b.WriteString(classifyPromptCore)
	if s := strings.TrimSpace(in.SessionSummary); s != "" {
		b.WriteString("\n\nSession summary (compressed earlier context):\n")
		b.WriteString(truncateRunes(s, planContextMaxSummary))
	}
	if d := strings.TrimSpace(in.RecentDialogue); d != "" {
		b.WriteString("\n\nRecent session dialogue (oldest first; current user turn is separate below):\n")
		b.WriteString(d)
	}
	fmt.Fprintf(&b, "\n\nLast turn domain: %s\nUser: %s", in.LastDomain, strings.TrimSpace(in.UserText))
	return b.String()
}
