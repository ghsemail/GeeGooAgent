package strategydev

import "strings"

// Buy score direction for soft annotation (would_intercept on buys only).
// JSON/API value: price_analysis_buy_score_direction = "normal" | "reverse".
type BuyScoreDirection string

const (
	// BuyScoreDirectionNormal: soft buy would_intercept when score < threshold (default).
	BuyScoreDirectionNormal BuyScoreDirection = "normal"
	// BuyScoreDirectionReverse: soft buy would_intercept when score > threshold; score == threshold does not intercept.
	BuyScoreDirectionReverse BuyScoreDirection = "reverse"
)

// ParseBuyScoreDirection parses workflow/diagnose option (unknown → normal).
func ParseBuyScoreDirection(raw string) BuyScoreDirection {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(BuyScoreDirectionReverse), "反向":
		return BuyScoreDirectionReverse
	default:
		return BuyScoreDirectionNormal
	}
}

// BuyScoreDirectionDisplayLabel is the Chinese label for filter stats / UI notes.
func BuyScoreDirectionDisplayLabel(d BuyScoreDirection) string {
	if d == BuyScoreDirectionReverse {
		return "反向"
	}
	return "现行"
}

// WouldInterceptTriggerByScoreMode applies score gate with soft vs hard rules.
// Soft (annotateOnly): sells never score-intercept; buys follow buyDir.
// Hard: unchanged legacy buy/sell thresholds (buyDir ignored).
func WouldInterceptTriggerByScoreMode(side string, score, threshold float64, buyDir BuyScoreDirection, annotateOnly bool) bool {
	if annotateOnly {
		if side == "sell" {
			return false
		}
		return wouldInterceptBuySoft(score, threshold, buyDir)
	}
	return ShouldInterceptTriggerByScore(side, score, threshold)
}

func wouldInterceptBuySoft(score, threshold float64, dir BuyScoreDirection) bool {
	score = NormalizePriceScore(score)
	threshold = NormalizePriceScore(threshold)
	switch dir {
	case BuyScoreDirectionReverse:
		if score == threshold {
			return false
		}
		return score > threshold
	default:
		return score < threshold
	}
}
