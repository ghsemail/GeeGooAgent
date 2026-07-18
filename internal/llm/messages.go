package llm

import "strings"

// SanitizeMessages enforces provider-friendly OpenAI chat ordering:
// merge consecutive user/assistant messages; keep tool sequences intact.
func SanitizeMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]Message, 0, len(messages))
	for _, m := range messages {
		if len(out) == 0 {
			out = append(out, m)
			continue
		}
		last := &out[len(out)-1]
		if m.Role == RoleTool {
			out = append(out, m)
			continue
		}
		if m.Role == RoleAssistant && len(last.ToolCalls) > 0 {
			out = append(out, m)
			continue
		}
		if m.Role == last.Role && (m.Role == RoleUser || m.Role == RoleAssistant) {
			mergeMessageContent(last, m)
			if len(m.ToolCalls) > 0 {
				last.ToolCalls = append(last.ToolCalls, m.ToolCalls...)
			}
			if rc := strings.TrimSpace(m.ReasoningContent); rc != "" {
				if strings.TrimSpace(last.ReasoningContent) != "" {
					last.ReasoningContent += "\n" + rc
				} else {
					last.ReasoningContent = rc
				}
			}
			continue
		}
		out = append(out, m)
	}
	return out
}

func mergeMessageContent(dst *Message, src Message) {
	a := strings.TrimSpace(dst.Content)
	b := strings.TrimSpace(src.Content)
	switch {
	case a == "":
		dst.Content = src.Content
	case b == "":
		return
	default:
		dst.Content = a + "\n\n" + b
	}
}
