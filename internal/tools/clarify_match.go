package tools

import (
	"fmt"
	"strings"
)

// MatchClarifyAnswer maps user text (A/B/C, 1/2, 第一, or full option) to a choice label.
func MatchClarifyAnswer(line string, choices []string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || len(choices) == 0 {
		return "", false
	}
	options := ClarifyDisplayOptions(choices)
	upper := strings.ToUpper(line)
	if len(upper) == 1 && upper[0] >= 'A' && int(upper[0]-'A') < len(options) {
		return options[int(upper[0]-'A')], true
	}
	if idx, ok := parseClarifyIndex(line, len(options)); ok {
		return options[idx], true
	}
	if idx, ok := parseChineseOrdinal(line, len(options)); ok {
		return options[idx], true
	}
	for _, opt := range options {
		if strings.EqualFold(strings.TrimSpace(opt), line) {
			return opt, true
		}
	}
	for _, opt := range choices {
		if strings.Contains(line, opt) || strings.Contains(opt, line) {
			return opt, true
		}
	}
	return "", false
}

func parseClarifyIndex(line string, n int) (int, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, false
	}
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil {
		return 0, false
	}
	if idx >= 1 && idx <= n {
		return idx - 1, true
	}
	return 0, false
}

var chineseOrdinals = []string{"第一", "第二", "第三", "第四", "第五", "第六", "第七", "第八"}

func parseChineseOrdinal(line string, n int) (int, bool) {
	line = strings.TrimSpace(line)
	for i, ord := range chineseOrdinals {
		if i >= n {
			break
		}
		if line == ord || strings.HasPrefix(line, ord) {
			return i, true
		}
	}
	return 0, false
}
