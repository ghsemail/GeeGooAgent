package verdict

import "strings"

// Confidence levels ordered from weakest to strongest.
const (
	ConfidenceReview = "review_required"
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

func normalizeConfidence(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ConfidenceHigh:
		return ConfidenceHigh
	case ConfidenceMedium, "med":
		return ConfidenceMedium
	case ConfidenceLow:
		return ConfidenceLow
	case ConfidenceReview, "review":
		return ConfidenceReview
	default:
		return ""
	}
}

func confidenceRank(v string) int {
	switch normalizeConfidence(v) {
	case ConfidenceReview:
		return 0
	case ConfidenceLow:
		return 1
	case ConfidenceMedium:
		return 2
	case ConfidenceHigh:
		return 3
	default:
		return 0
	}
}

func minConfidence(a, b string) string {
	a = normalizeConfidence(a)
	b = normalizeConfidence(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if confidenceRank(a) <= confidenceRank(b) {
		return a
	}
	return b
}

func downgradeConfidence(v string) string {
	switch normalizeConfidence(v) {
	case ConfidenceHigh:
		return ConfidenceMedium
	case ConfidenceMedium:
		return ConfidenceLow
	default:
		return ConfidenceReview
	}
}

func upgradeConfidence(v string) string {
	switch normalizeConfidence(v) {
	case ConfidenceLow:
		return ConfidenceMedium
	case ConfidenceMedium:
		return ConfidenceHigh
	default:
		return v
	}
}

func normalizeResult(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "long", "bullish", "positive", "buy":
		return "long"
	case "short", "bearish", "negative", "sell":
		return "short"
	case "neutral", "hold", "mixed", "flat":
		return "neutral"
	default:
		return ""
	}
}

func resultsConflict(a, b string) bool {
	a = normalizeResult(a)
	b = normalizeResult(b)
	if a == "" || b == "" || a == "neutral" || b == "neutral" {
		return false
	}
	return a != b
}
