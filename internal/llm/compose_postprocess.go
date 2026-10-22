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
	last := -1
	for i, r := range runes {
		if isSentenceEndRune(r) {
			last = i
		}
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

func isSentenceEndRune(r rune) bool {
	switch r {
	case '。', '！', '？', '；', '.', '!', '?':
		return true
	default:
		return false
	}
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
