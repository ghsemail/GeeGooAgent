package strategydev

import (
	"fmt"
	"strings"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// DiagnoseLabMode reports whether signal-diagnose should run lab enrichments for this request.
func DiagnoseLabMode(labMode bool) bool {
	return labMode
}

// LabEpisodeCardSummary is a one-line lab conclusion on a diagnose card (episode or trigger).
type LabEpisodeCardSummary struct {
	KlinePattern  string   `json:"kline_pattern,omitempty"`
	PriceAnalysis string   `json:"price_analysis,omitempty"`
	DetailRef     string   `json:"detail_ref,omitempty"`
	Degraded      []string `json:"degraded,omitempty"`
}

// LabEpisodeDetail is full lab payload for modal drill-down.
type LabEpisodeDetail struct {
	Ref           string                 `json:"ref"`
	Side          string                 `json:"side"`
	BarIndex      int                    `json:"bar_index"`
	Time          string                 `json:"time,omitempty"`
	Close         float64                `json:"close,omitempty"`
	KlinePattern  *LabKlinePatternDetail `json:"kline_pattern,omitempty"`
	PriceAnalysis *PriceAnalysis         `json:"price_analysis,omitempty"`
	Degraded      []string               `json:"degraded,omitempty"`
}

// LabKlinePatternDetail holds MTF tables and hit lists for one anchor bar.
type LabKlinePatternDetail struct {
	Summary  string                            `json:"summary,omitempty"`
	Bias     *slots.KlinePatternBias           `json:"bias,omitempty"`
	Patterns []slots.KlinePatternHit           `json:"patterns,omitempty"`
	ThreeBar *slots.KlinePatternThreeBarWindow `json:"three_bar,omitempty"`
	MTF      *slots.KlinePatternMTFContext     `json:"mtf,omitempty"`
}

// LabCardSummaryLine formats direction/score/key pattern for UI subtitle.
func LabCardSummaryLine(kind, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	switch kind {
	case "kline":
		return "形态 · " + text
	case "price":
		return "价格 · " + text
	default:
		return text
	}
}

// LabFlowDetailRef opens run-level lab detail when there are no trigger rows.
const LabFlowDetailRef = "flow:0"

// LabDetailRef builds stable ref ids (buy:3 / sell:1).
func LabDetailRef(side string, barIndex int) string {
	side = strings.ToLower(strings.TrimSpace(side))
	if side == "" {
		side = "buy"
	}
	return fmt.Sprintf("%s:%d", side, barIndex)
}

// ParseLabDetailRef parses side and bar index from detail ref.
func ParseLabDetailRef(ref string) (side string, barIndex int, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == LabFlowDetailRef {
		return "flow", 0, true
	}
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	side = strings.ToLower(strings.TrimSpace(parts[0]))
	if side == "flow" && strings.TrimSpace(parts[1]) == "0" {
		return "flow", 0, true
	}
	if side != "buy" && side != "sell" {
		return "", 0, false
	}
	var idx int
	if _, err := fmt.Sscanf(parts[1], "%d", &idx); err != nil || idx < 0 {
		return "", 0, false
	}
	return side, idx, true
}

// LightKlineSummaryAtIndex uses signal-TF pattern_hits only (no MTF probe).
func LightKlineSummaryAtIndex(patternHits []any, idx int) string {
	hits := kp.FilteredKlinePatternsAtIndex(patternHits, idx)
	if len(hits) == 0 {
		return "无明显形态"
	}
	return slots.KlinePatternSummary(hits)
}

// PriceAnalysisSummaryLine compresses multi-TF price analysis to one card line.
func PriceAnalysisSummaryLine(pa *PriceAnalysis) string {
	if pa == nil {
		return ""
	}
	if s := strings.TrimSpace(pa.Summary); s != "" {
		return s
	}
	if pa.PriceScore != 0 || pa.ScoreReason != "" {
		if pa.ScoreReason != "" {
			return fmt.Sprintf("综合分 %.1f · %s", pa.PriceScore, strings.TrimSpace(pa.ScoreReason))
		}
		return fmt.Sprintf("综合分 %.1f", pa.PriceScore)
	}
	if len(pa.Rows) > 0 {
		return strings.TrimSpace(pa.Rows[0].Conclusion)
	}
	return ""
}
