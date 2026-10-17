package slots

import (
	"fmt"
	"strings"
)

// KlinePatternHit is one documented candlestick pattern on a bar (display-only, not a trade gate).
type KlinePatternHit struct {
	Name      string `json:"name"`
	Direction int    `json:"direction"` // 1 bullish, -1 bearish, 2 doji/cross
	LabelCN   string `json:"label_cn,omitempty"`
}

// KlinePatternSummary renders short Chinese labels for cards and markdown.
func KlinePatternSummary(hits []KlinePatternHit) string {
	if len(hits) == 0 {
		return ""
	}
	hasBull, hasBear := false, false
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		switch h.Direction {
		case 1:
			hasBull = true
		case -1:
			hasBear = true
		}
		label := strings.TrimSpace(h.LabelCN)
		if label == "" {
			label = h.Name
		}
		dir := klinePatternDirectionCN(h.Direction)
		if dir != "" {
			parts = append(parts, fmt.Sprintf("%s(%s)", label, dir))
		} else {
			parts = append(parts, label)
		}
	}
	prefix := ""
	if hasBull && hasBear {
		prefix = "多空并存："
	}
	return prefix + strings.Join(parts, "、")
}

func klinePatternDirectionCN(dir int) string {
	switch dir {
	case 1:
		return "看多"
	case -1:
		return "看空"
	case 2:
		return "十字类"
	default:
		return ""
	}
}
