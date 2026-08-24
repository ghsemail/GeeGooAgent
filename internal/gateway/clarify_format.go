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
		return q + "\n\n" + FormatClarifyInquiryHint()
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
	b.WriteString("\n回复字母（如 A）或输入你的选择；回复「跳过」可暂不回答。")
	return b.String()
}

// FormatClarifyInquiryHint guides open-ended clarify on IM channels.
func FormatClarifyInquiryHint() string {
	return "请直接回复你的答案（尽量具体，便于我继续处理）。\n回复「跳过」可暂不回答。"
}

// FormatClarifyCustomPrompt asks for free-text after the user picks Other.
func FormatClarifyCustomPrompt() string {
	return "请直接输入你的选择（回复「跳过」可暂不回答）："
}

// IsClarifySkip reports whether IM text means skip clarify.
func IsClarifySkip(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	switch text {
	case "skip", "pass", "cancel", "none", "n/a":
		return true
	}
	return text == "跳过" || text == "略过" || text == "取消"
}

// IsOtherClarifySelection reports whether text selects the appended Other option.
func IsOtherClarifySelection(text string, choices []string) bool {
	if len(choices) == 0 {
		return false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	display := tools.ClarifyDisplayOptions(choices)
	otherIdx := len(display) - 1
	if len(text) == 1 && text[0] >= 'a' && text[0] <= 'z' {
		text = strings.ToUpper(text)
	}
	if len(text) == 1 && text[0] >= 'A' && text[0] <= 'Z' {
		return int(text[0]-'A') == otherIdx
	}
	if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(display) {
		return n-1 == otherIdx
	}
	if strings.EqualFold(text, tools.ClarifyOtherLabel) {
		return true
	}
	if strings.Contains(text, "其他") && len(text) <= 12 {
		return true
	}
	return false
}

// ParseClarifyReply maps IM text to a clarify answer.
func ParseClarifyReply(text string, choices []string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	if IsClarifySkip(text) {
		return "", false
	}
	if len(choices) == 0 {
		return text, true
	}
	if IsOtherClarifySelection(text, choices) {
		return "", false
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
	if lower == "other" {
		return "", false
	}
	return text, true
}
