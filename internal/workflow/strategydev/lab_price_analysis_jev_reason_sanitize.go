package strategydev

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var (
	labJEVQuotedChineseRE     = regexp.MustCompile(`["「]([^"」\n]{6,}?)["」]`)
	labJEVUnclosedThinkRE     = regexp.MustCompile(`(?is)<(?:redacted_thinking|think)>(.*)$`)
	labJEVDanglingReasonStartRE = regexp.MustCompile(`^[\d.\s]+[）)]`)
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
	fallback := labJEVReasonTrim(llm.PublicAnswerContent(raw))
	if labJEVReasonLooksValid(fallback) {
		return fallback
	}
	if c := labJEVExtractChineseSentences(mixed); c != "" {
		return labJEVReasonTrim(c)
	}
	return fallback
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
		candidate := labJEVReasonReplaceProductNames(strings.TrimSpace(m[1]))
		if !labJEVReasonQuotedCandidateOK(candidate) {
			continue
		}
		if len([]rune(candidate)) > len([]rune(best)) {
			best = candidate
		}
	}
	return best
}

func labJEVTrimToReasonStart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if i := strings.Index(s, "JEV"); i >= 0 {
		return strings.TrimSpace(s[i:])
	}
	for i, r := range s {
		if unicode.Is(unicode.Han, r) {
			return strings.TrimSpace(s[i:])
		}
	}
	return s
}

func labJEVReasonRuneAllowedInChineseSpan(r rune) bool {
	if r == '−' || r == '–' || r == '+' {
		return true
	}
	return unicode.Is(unicode.Han, r) || unicode.IsPunct(r) || unicode.IsNumber(r) ||
		r == ' ' || r == '%' || r == '.' || unicode.IsLetter(r)
}

func labJEVReasonSummaryStartOK(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if labJEVDanglingReasonStartRE.MatchString(s) {
		return false
	}
	if strings.HasPrefix(s, "JEV") {
		return true
	}
	r0, _ := utf8.DecodeRuneInString(s)
	return unicode.Is(unicode.Han, r0)
}

func labJEVReasonSubstantialChinese(s string) bool {
	s = strings.TrimSpace(s)
	var cjk, latin int
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			cjk++
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}
	if cjk < 8 {
		return false
	}
	if strings.HasPrefix(s, "JEV") {
		return true
	}
	return cjk >= latin
}

func labJEVReasonQuotedCandidateOK(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !labJEVReasonSummaryStartOK(s) {
		return false
	}
	return llm.IsMostlyChinese(s) || labJEVReasonSubstantialChinese(s)
}

func labJEVExtractChineseSentences(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var parts []string
	for _, line := range strings.Split(s, "\n") {
		line = labJEVTrimToReasonStart(strings.TrimSpace(line))
		if line == "" || labJEVReasonLooksLikePlanning(line) {
			continue
		}
		if labJEVReasonLooksValid(line) {
			parts = append(parts, line)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "")
	}
	// Fallback: longest mostly-Chinese span; keep Latin indicator tokens (e.g. Force Index).
	s = labJEVTrimToReasonStart(s)
	var buf strings.Builder
	var best strings.Builder
	flush := func() {
		if labJEVReasonLooksValid(buf.String()) && buf.Len() > best.Len() {
			best.Reset()
			best.WriteString(buf.String())
		}
		buf.Reset()
	}
	for _, r := range s {
		if labJEVReasonRuneAllowedInChineseSpan(r) {
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
	if !labJEVReasonSummaryStartOK(s) {
		return false
	}
	return llm.IsMostlyChinese(s) || labJEVReasonSubstantialChinese(s)
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
	if !labJEVReasonSummaryStartOK(s) {
		return ""
	}
	s = labJEVReasonReplaceProductNames(s)
	const maxRunes = 280
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(runes[:maxRunes]) + "…")
}

// labJEVReasonReplaceProductNames maps vendor labels to trader-facing Chinese.
func labJEVReasonReplaceProductNames(s string) string {
	repl := strings.NewReplacer(
		"System One 给出", "JEV 判定为",
		"system one 给出", "JEV 判定为",
		"System One", "JEV",
		"system one", "JEV",
		"SystemOne", "JEV",
	)
	return repl.Replace(s)
}
