package strategydev

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabJEVUnifiedDirection is the display mirror of price_analysis_jev (no local weighted fuse).
type LabJEVUnifiedDirection string

const (
	LabJEVUnifiedBull    LabJEVUnifiedDirection = "bull"
	LabJEVUnifiedBear    LabJEVUnifiedDirection = "bear"
	LabJEVUnifiedUnclear LabJEVUnifiedDirection = "unclear"
)

// LabJEVUnifiedBreakdown holds human-readable snippets for traders / FE.
type LabJEVUnifiedBreakdown struct {
	Structure string `json:"structure"`
	Pattern   string `json:"pattern"`
}

// LabJEVUnified mirrors the single PA+pattern JEV read for legacy FE fields (no 0.7/0.3 synthesis).
type LabJEVUnified struct {
	Direction  LabJEVUnifiedDirection `json:"direction"`
	Confidence float64                `json:"confidence"`
	Conflict   bool                   `json:"conflict"`
	Breakdown  LabJEVUnifiedBreakdown `json:"breakdown"`
}

// ApplyLabJEVUnified sets jev_unified from price_analysis_jev only (deprecated weighted path removed).
func ApplyLabJEVUnified(out *LabEpisodeDetail) {
	if out == nil || out.PriceAnalysisJEV == nil || !out.PriceAnalysisJEV.OK {
		return
	}
	jev := out.PriceAnalysisJEV
	dir := paJEVLabelToDirection(jev.RecommendedChoice)
	conf := jevChoiceConfidence(jev)
	structSum := strings.TrimSpace(jev.RecommendedReason)
	if structSum == "" {
		structSum = "结构+形态 JEV：" + jev.RecommendedChoice
	}
	patSum := patternSummaryText(out)
	out.JEVUnified = &LabJEVUnified{
		Direction:  dir,
		Confidence: conf,
		Conflict:   labJEVDisplayConflict(out, jev.RecommendedChoice),
		Breakdown: LabJEVUnifiedBreakdown{
			Structure: structSum,
			Pattern:   patSum,
		},
	}
}

func paJEVLabelToDirection(label string) LabJEVUnifiedDirection {
	switch strings.TrimSpace(label) {
	case "看多":
		return LabJEVUnifiedBull
	case "看空":
		return LabJEVUnifiedBear
	default:
		return LabJEVUnifiedUnclear
	}
}

func jevChoiceConfidence(jev *PriceAnalysisJEV) float64 {
	if jev == nil || len(jev.Choices) == 0 {
		return 0
	}
	idx := jev.RecommendedIndex
	if idx < 0 || idx >= len(jev.Choices) {
		idx = 0
	}
	return clampUnitInterval(jev.Choices[idx].Probability)
}

func patternSummaryText(out *LabEpisodeDetail) string {
	if out.KlinePattern != nil && strings.TrimSpace(out.KlinePattern.Summary) != "" {
		return strings.TrimSpace(out.KlinePattern.Summary)
	}
	return strings.TrimSpace(out.KlinePatternSummary)
}

func patternBiasForDetail(out *LabEpisodeDetail) *slots.KlinePatternBias {
	if out.KlinePatternBias != nil {
		return out.KlinePatternBias
	}
	if out.KlinePattern != nil && out.KlinePattern.Bias != nil {
		return out.KlinePattern.Bias
	}
	return nil
}

// labJEVDisplayConflict is informational for FE only; decision is solely price_analysis_jev.
func labJEVDisplayConflict(out *LabEpisodeDetail, jevChoice string) bool {
	bias := patternBiasForDetail(out)
	if bias == nil {
		return false
	}
	b := strings.TrimSpace(bias.Bias)
	if b == "" || b == "中性" {
		return false
	}
	return b != strings.TrimSpace(jevChoice) && strings.TrimSpace(jevChoice) != "不明确"
}

func clampUnitInterval(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
