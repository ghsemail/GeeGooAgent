package fts

import (
	"regexp"
	"strings"
)

var tokenRe = regexp.MustCompile(`[a-zA-Z0-9]{2,}`)

// BuildQuery tokenizes text into a PostgreSQL tsquery OR chain (Waku FTS parity).
func BuildQuery(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	seen := map[string]struct{}{}
	var parts []string
	for _, tok := range tokenRe.FindAllString(text, -1) {
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		parts = append(parts, tok)
	}
	return strings.Join(parts, " | ")
}
