package llm

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	mdHeadingMissingSpaceRE = regexp.MustCompile(`(?m)^(#{1,6})([^\s#\n])`)
	mdHeadingBodyGlueRE     = regexp.MustCompile(`(?m)^(#{1,6}\s+\p{Han}+?(?:评价|报告|结论|摘要))(\p{Han})`)
)

// NormalizeComposeMarkdown fixes common model glitches in diag_compose output.
func NormalizeComposeMarkdown(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = mdHeadingMissingSpaceRE.ReplaceAllString(s, "$1 $2")
	s = mdHeadingBodyGlueRE.ReplaceAllString(s, "$1\n\n$2")
	return strings.TrimSpace(s)
}

// TruncateToLastCompleteSentence trims trailing incomplete fragment (deadline mid-sentence).
func TruncateToLastCompleteSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	last := lastChineseSentenceEnd(runes)
	if last < 0 {
		last = lastASCIISentenceEnd(runes)
	}
	if last < 0 {
		return s
	}
	out := strings.TrimSpace(string(runes[:last+1]))
	if out == "" {
		return s
	}
	return out
}

func lastChineseSentenceEnd(runes []rune) int {
	last := -1
	for i, r := range runes {
		switch r {
		case '。', '！', '？', '；':
			last = i
		}
	}
	return last
}

func lastASCIISentenceEnd(runes []rune) int {
	last := -1
	for i, r := range runes {
		switch r {
		case '!', '?':
			last = i
		case '.':
			if asciiPeriodEndsSentence(runes, i) {
				last = i
			}
		}
	}
	return last
}

func asciiPeriodEndsSentence(runes []rune, i int) bool {
	if i <= 0 {
		return false
	}
	prev := runes[i-1]
	if prev >= 'A' && prev <= 'Z' {
		// Ticker suffix AAPL. before more Chinese — not a sentence end.
		if i+1 < len(runes) && unicode.Is(unicode.Han, runes[i+1]) {
			return false
		}
		if i+1 < len(runes) && ((runes[i+1] >= 'a' && runes[i+1] <= 'z') || (runes[i+1] >= 'A' && runes[i+1] <= 'Z')) {
			return false
		}
	}
	if unicode.Is(unicode.Han, prev) {
		return true
	}
	if i+1 >= len(runes) {
		return true
	}
	next := runes[i+1]
	return next == '\n' || next == ' ' || unicode.Is(unicode.Han, next)
}

// IsMostlyChinese reports whether runes are predominantly CJK (visible answer quality check).
func IsMostlyChinese(s string) bool {
	var cjk, other int
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			cjk++
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			other++
		}
	}
	return cjk > 0 && cjk >= other
}
