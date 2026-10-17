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

// KlinePatternBias is a display-only aggregate from whitelisted pattern directions.
type KlinePatternBias struct {
	Score     float64 `json:"score"`
	Bias      string  `json:"bias"`       // 看多 | 看空 | 中性
	BiasColor string  `json:"bias_color"` // green | red | gray
}

// KlinePatternBarWindowEntry is one bar in the prev/anchor/next window.
type KlinePatternBarWindowEntry struct {
	Role                string            `json:"role"` // prev | anchor | next
	Index               int               `json:"index,omitempty"`
	Omitted             bool              `json:"omitted,omitempty"`
	OmitReason          string            `json:"omit_reason,omitempty"` // before_series_start | after_series_end
	Time                string            `json:"time,omitempty"`
	Open                float64           `json:"open,omitempty"`
	High                float64           `json:"high,omitempty"`
	Low                 float64           `json:"low,omitempty"`
	Close               float64           `json:"close,omitempty"`
	KlinePatterns       []KlinePatternHit `json:"kline_patterns,omitempty"`
	KlinePatternSummary string            `json:"kline_pattern_summary,omitempty"`
	KlinePatternBias    *KlinePatternBias `json:"kline_pattern_bias,omitempty"`
}

// KlinePatternThreeBarWindow is prev + trigger anchor + next on the signal timeframe.
type KlinePatternThreeBarWindow struct {
	AnchorIndex int                          `json:"anchor_index"`
	Bars        []KlinePatternBarWindowEntry `json:"bars"`
	WindowNote  string                       `json:"window_note,omitempty"`
	Bias        *KlinePatternBias            `json:"bias,omitempty"` // anchor-bar primary score
	WindowBias  *KlinePatternBias            `json:"window_bias,omitempty"`
}

// KlinePatternMTFEntry is higher-TF background context aligned to the trigger instant.
type KlinePatternMTFEntry struct {
	Frequency           string                    `json:"frequency"`
	Label               string                    `json:"label"`
	DisplayOnly         bool                      `json:"display_only"`
	AlignedIndex        int                       `json:"aligned_index,omitempty"`
	AlignedTime         string                    `json:"aligned_time,omitempty"`
	AlignNote           string                    `json:"align_note,omitempty"`
	ThreeBar            *KlinePatternThreeBarWindow `json:"three_bar,omitempty"`
	Bias                *KlinePatternBias         `json:"bias,omitempty"`
	FetchOK             bool                      `json:"fetch_ok,omitempty"`
}

// KlinePatternMTFContext groups coarser timeframe views (never replaces signal-TF judgment).
type KlinePatternMTFContext struct {
	SignalFrequency string                 `json:"signal_frequency"`
	Note            string                 `json:"note,omitempty"`
	PrimaryBias     *KlinePatternBias      `json:"primary_bias,omitempty"`
	Timeframes      []KlinePatternMTFEntry `json:"timeframes,omitempty"`
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

// ScoreKlinePatternHits sums direction weights (+1 / -1 / 0) for display bias.
func ScoreKlinePatternHits(hits []KlinePatternHit) KlinePatternBias {
	var score float64
	hasBull, hasBear := false, false
	for _, h := range hits {
		switch h.Direction {
		case 1:
			score += 1
			hasBull = true
		case -1:
			score -= 1
			hasBear = true
		}
	}
	bias := "中性"
	color := "gray"
	if score > 0 {
		bias = "看多"
		color = "green"
	} else if score < 0 {
		bias = "看空"
		color = "red"
	}
	if hasBull && hasBear && score == 0 {
		bias = "中性"
		color = "gray"
	}
	return KlinePatternBias{Score: score, Bias: bias, BiasColor: color}
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
