package strategydev

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var (
	labJEVQuotedChineseRE = regexp.MustCompile(`["「]([^"」\n]{6,}?)["」]`)
	labJEVUnclosedThinkRE = regexp.MustCompile(`(?is)<(?:redacted_thinking|think)>(.*)$`)
)

// sanitizeLabJEVReasonSummary strips inline thinking / English planning from synthesis LLM output.
func sanitizeLabJEVReasonSummary(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if v := llm.PublicAnswerContent(raw); labJEVReasonLooksValid(v) {
		return labJEVReasonTrim(v)
	}
	mixed := labJEVExtractThinkBody(raw)
	if q := labJEVExtractQuotedChinese(mixed); labJEVReasonLooksValid(q) {
		return labJEVReasonTrim(q)
	}
	if c := labJEVExtractChineseSentences(mixed); labJEVReasonLooksValid(c) {
		return labJEVReasonTrim(c)
	}
	if v := llm.PublicAnswerContent(mixed); labJEVReasonLooksValid(v) {
		return labJEVReasonTrim(v)
	}
	return labJEVReasonTrim(llm.PublicAnswerContent(raw))
}

func labJEVExtractThinkBody(s string) string {
	s = strings.TrimSpace(s)
	visible, extracted := llm.SplitInlineThinking(s)
	if strings.TrimSpace(visible) != "" {
		return strings.TrimSpace(visible)
	}
	if strings.TrimSpace(extracted) != "" {
		return strings.TrimSpace(extracted)
	}
	if m := labJEVUnclosedThinkRE.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return s
}

func labJEVExtractQuotedChinese(s string) string {
	var best string
	for _, m := range labJEVQuotedChineseRE.FindAllStringSubmatch(s, -1) {
		if len(m) < 2 {
			continue
		}
		candidate := strings.TrimSpace(m[1])
		if !llm.IsMostlyChinese(candidate) {
			continue
		}
		if len([]rune(candidate)) > len([]rune(best)) {
			best = candidate
		}
	}
	return best
}

func labJEVExtractChineseSentences(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var parts []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || labJEVReasonLooksLikePlanning(line) {
			continue
		}
		if llm.IsMostlyChinese(line) {
			parts = append(parts, line)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "")
	}
	// Fallback: longest mostly-Chinese substring between punctuation.
	var buf strings.Builder
	var best strings.Builder
	flush := func() {
		if llm.IsMostlyChinese(buf.String()) && buf.Len() > best.Len() {
			best.Reset()
			best.WriteString(buf.String())
		}
		buf.Reset()
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.IsPunct(r) || unicode.IsNumber(r) ||
			r == ' ' || r == '%' || r == '.' {
			buf.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return strings.TrimSpace(best.String())
}

func labJEVReasonLooksValid(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.Contains(strings.ToLower(s), "<think") {
		return false
	}
	if labJEVReasonLooksLikePlanning(s) {
		return false
	}
	return llm.IsMostlyChinese(s)
}

func labJEVReasonLooksLikePlanning(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "we need ") ||
		strings.Contains(lower, "no markdown") ||
		strings.Contains(lower, "1-2 sentences") ||
		strings.Contains(lower, "without changing") {
		return true
	}
	return !llm.IsMostlyChinese(s) && strings.Contains(lower, "answer chinese")
}

func labJEVReasonTrim(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	const maxRunes = 280
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(runes[:maxRunes]) + "…")
}
