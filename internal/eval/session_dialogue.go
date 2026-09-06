package eval

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// LastAssistantReply returns the latest non-empty assistant message in the session.
func LastAssistantReply(chat *chatsession.ChatSession) string {
	if chat == nil {
		return ""
	}
	for i := len(chat.Messages) - 1; i >= 0; i-- {
		msg := chat.Messages[i]
		if msg.Role != llm.RoleAssistant {
			continue
		}
		if strings.TrimSpace(msg.Content) != "" {
			return strings.TrimSpace(msg.Content)
		}
	}
	return ""
}

// DialogueSnapshotFromSession builds a readable user/assistant transcript for eval UI.
func DialogueSnapshotFromSession(chat *chatsession.ChatSession) []DialogueSnapshotTurn {
	if chat == nil {
		return nil
	}
	out := make([]DialogueSnapshotTurn, 0, len(chat.Messages))
	for _, msg := range chat.Messages {
		switch msg.Role {
		case llm.RoleUser:
			text := strings.TrimSpace(msg.Content)
			if text == "" || strings.HasPrefix(text, "本会话 Tool 活动") {
				continue
			}
			out = append(out, DialogueSnapshotTurn{Role: "user", Text: text})
		case llm.RoleAssistant:
			text := strings.TrimSpace(msg.Content)
			if text == "" {
				continue
			}
			out = append(out, DialogueSnapshotTurn{Role: "assistant", Text: text})
		default:
			continue
		}
	}
	return out
}

// DialogueContextSummary returns a compact prior-turn summary for the judge prompt.
func DialogueContextSummary(snapshot []DialogueSnapshotTurn, maxTurns int) string {
	if len(snapshot) == 0 {
		return ""
	}
	if maxTurns <= 0 {
		maxTurns = 6
	}
	start := 0
	if len(snapshot) > maxTurns {
		start = len(snapshot) - maxTurns
	}
	var b strings.Builder
	for _, turn := range snapshot[start:] {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		switch turn.Role {
		case "user":
			b.WriteString("用户：")
		case "assistant":
			b.WriteString("助手：")
		default:
			b.WriteString(turn.Role + "：")
		}
		b.WriteString(truncateRunes(turn.Text, 400))
	}
	return b.String()
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
