package runtime

import (
	"fmt"
	"strings"
)

// MergedPanelFrequency returns the effective K-line frequency from inbound + sticky panel ops.
func MergedPanelFrequency(session *Session) string {
	if session == nil {
		return ""
	}
	merged := MergedPanelOps(session)
	if merged == nil {
		return ""
	}
	return strings.TrimSpace(merged.Frequency)
}

// PanelOpsTurnHint is a short LLM-visible block when Monday signal_diagnose panel ops apply this turn.
func PanelOpsTurnHint(session *Session) string {
	if session == nil {
		return ""
	}
	merged := MergedPanelOps(session)
	if merged == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Monday signal_diagnose panel (this turn)\n")
	wrote := false
	if freq := strings.TrimSpace(merged.Frequency); freq != "" {
		fmt.Fprintf(&b, "- **frequency (confirmed)**: %s — treat as the probe/diagnose K-line period; **do not** call clarify to ask for K-line period.\n", freq)
		wrote = true
	}
	if merged.MonthsBack > 0 {
		fmt.Fprintf(&b, "- months_back: %d\n", merged.MonthsBack)
		wrote = true
	}
	if label := strings.TrimSpace(merged.PanelStrategyLabel); label != "" {
		fmt.Fprintf(&b, "- strategy: %s\n", label)
		wrote = true
	}
	if len(merged.BuySignal) > 0 || len(merged.SellSignal) > 0 {
		b.WriteString("- buy/sell signals are pre-filled from the panel; server injects them into probe/diagnose.\n")
		wrote = true
	}
	if !wrote {
		return ""
	}
	b.WriteString("- Mention the panel frequency briefly in your reply when you start probe (e.g. 「按面板 60m」).\n")
	return strings.TrimSpace(b.String())
}

// IsKLineFrequencyClarify reports whether a clarify prompt is asking the user to pick a K-line timeframe.
func IsKLineFrequencyClarify(question string, choices []string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return false
	}
	asksPeriod := strings.Contains(q, "周期") ||
		strings.Contains(q, "k线") ||
		strings.Contains(q, "k 线") ||
		strings.Contains(q, "frequency") ||
		strings.Contains(q, "频率")
	if !asksPeriod {
		return false
	}
	if len(choices) == 0 {
		return true
	}
	for _, c := range choices {
		cl := strings.ToLower(strings.TrimSpace(c))
		if cl == "" {
			continue
		}
		if strings.Contains(cl, "60m") || strings.Contains(cl, "daily") ||
			strings.Contains(cl, "15m") || strings.Contains(cl, "5m") ||
			strings.Contains(cl, "分钟") || strings.Contains(cl, "日线") {
			return true
		}
	}
	return asksPeriod
}

// TryAnswerFrequencyFromPanelOps auto-answers a K-line frequency clarify when panel ops already set frequency.
func TryAnswerFrequencyFromPanelOps(session *Session, question string, choices []string) (string, bool) {
	if !IsKLineFrequencyClarify(question, choices) {
		return "", false
	}
	freq := MergedPanelFrequency(session)
	if freq == "" {
		return "", false
	}
	if len(choices) > 0 {
		for _, c := range choices {
			c = strings.TrimSpace(c)
			if strings.EqualFold(c, freq) {
				return c, true
			}
			if strings.Contains(strings.ToLower(c), strings.ToLower(freq)) {
				return c, true
			}
		}
	}
	return freq, true
}
