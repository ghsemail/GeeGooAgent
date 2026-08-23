package gateway

import (
	"strconv"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// FormatClarifyMessage renders a Feishu-friendly option list for clarify.
func FormatClarifyMessage(question string, choices []string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		q = "请确认"
	}
	if len(choices) == 0 {
		return q + "\n\n请直接回复你的答案。"
	}
	display := tools.ClarifyDisplayOptions(choices)
	var b strings.Builder
	b.WriteString(q)
	b.WriteString("\n\n")
	for i, choice := range display {
		b.WriteString(tools.ClarifyChoiceLabel(i))
		b.WriteString(". ")
		b.WriteString(choice)
		b.WriteByte('\n')
	}
	b.WriteString("\n回复字母（如 A）或输入你的选择。")
	return b.String()
}

// ParseClarifyReply maps IM text to a clarify answer.
func ParseClarifyReply(text string, choices []string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	if len(choices) == 0 {
		return text, true
	}
	display := tools.ClarifyDisplayOptions(choices)
	lower := strings.ToLower(text)
	if len(text) == 1 && text[0] >= 'a' && text[0] <= 'z' {
		text = strings.ToUpper(text)
	}
	if len(text) == 1 && text[0] >= 'A' && text[0] <= 'Z' {
		idx := int(text[0] - 'A')
		if idx >= 0 && idx < len(display) {
			picked := display[idx]
			if picked == tools.ClarifyOtherLabel {
				return "", false
			}
			return picked, true
		}
	}
	if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(display) {
		picked := display[n-1]
		if picked == tools.ClarifyOtherLabel {
			return "", false
		}
		return picked, true
	}
	for _, choice := range display {
		if strings.EqualFold(text, choice) {
			if choice == tools.ClarifyOtherLabel {
				return "", false
			}
			return choice, true
		}
	}
	if lower == "other" || strings.Contains(text, "其他") {
		return "", false
	}
	return text, true
}
