package textutil

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// OneLine collapses whitespace and truncates to n runes.
func OneLine(s string, n int) string {
	return memory.OneLine(s, n)
}

// PlainSummary extracts plain text from markdown and truncates.
func PlainSummary(markdown string, n int) string {
	lines := strings.Split(markdown, "\n")
	plain := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#- ")
		if line != "" {
			plain = append(plain, line)
		}
	}
	return OneLine(strings.Join(plain, " "), n)
}
