package strategydev

import (
	"fmt"
	"math"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabJEVUnifiedDirection is the display-only composite read from PA-JEV + local kline patterns.
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

// LabJEVUnified is a display-only synthesis of structure (price_analysis_jev) and kline pattern bias.
type LabJEVUnified struct {
	Direction  LabJEVUnifiedDirection `json:"direction"`
	Confidence float64                `json:"confidence"`
	Conflict   bool                   `json:"conflict"`
	Breakdown  LabJEVUnifiedBreakdown `json:"breakdown"`
}

type jevUnifiedLeg struct {
	dir        LabJEVUnifiedDirection
	confidence float64
	summary    string
	present    bool
}

// ApplyLabJEVUnified computes jev_unified after PA-JEV and kline pattern fields are populated.
func ApplyLabJEVUnified(out *LabEpisodeDetail) {
	if out == nil {
		return
	}
	structure := structureLegFromPAJEV(out.PriceAnalysisJEV)
	pattern := patternLegFromKline(out)
	out.JEVUnified = synthesizeLabJEVUnified(structure, pattern)
}

func structureLegFromPAJEV(jev *PriceAnalysisJEV) jevUnifiedLeg {
	if jev == nil || !jev.OK {
		sum := "结构 JEV 不可用"
		if jev != nil && strings.TrimSpace(jev.Error) != "" {
			sum = "结构 JEV 不可用：" + strings.TrimSpace(jev.Error)
		}
		return jevUnifiedLeg{dir: LabJEVUnifiedUnclear, confidence: 0, summary: sum, present: false}
	}
	dir := paJEVLabelToDirection(jev.RecommendedChoice)
	conf := structureConfidenceFromPAJEV(jev)
	sum := strings.TrimSpace(jev.RecommendedReason)
	if sum == "" {
		sum = fmt.Sprintf("结构倾向%s", jev.RecommendedChoice)
	}
	return jevUnifiedLeg{dir: dir, confidence: conf, summary: sum, present: true}
}

func structureConfidenceFromPAJEV(jev *PriceAnalysisJEV) float64 {
	if jev == nil || len(jev.Choices) == 0 {
		return 0
	}
	idx := jev.RecommendedIndex
	if idx < 0 || idx >= len(jev.Choices) {
		idx = 0
	}
	return clampUnitInterval(jev.Choices[idx].Probability)
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

func patternLegFromKline(out *LabEpisodeDetail) jevUnifiedLeg {
	if out == nil {
		return jevUnifiedLeg{present: false, summary: "无本地形态数据"}
	}
	summary := patternSummaryText(out)
	bias := patternBiasForUnified(out)
	hasData := bias != nil || summary != "" || len(out.KlinePatterns) > 0
	if !hasData {
		return jevUnifiedLeg{present: false, summary: "无本地形态数据"}
	}
	dir := klineBiasToDirection(bias)
	conf := patternConfidenceFromBias(bias, out.KlinePatterns)
	if summary == "" && bias != nil {
		summary = strings.TrimSpace(bias.Bias)
		if bias.Score != 0 {
			summary = fmt.Sprintf("%s（得分 %.2f）", summary, bias.Score)
		}
	}
	if summary == "" {
		summary = "无明显形态"
	}
	return jevUnifiedLeg{dir: dir, confidence: conf, summary: summary, present: true}
}

func patternSummaryText(out *LabEpisodeDetail) string {
	if out.KlinePattern != nil && strings.TrimSpace(out.KlinePattern.Summary) != "" {
		return strings.TrimSpace(out.KlinePattern.Summary)
	}
	return strings.TrimSpace(out.KlinePatternSummary)
}

func patternBiasForUnified(out *LabEpisodeDetail) *slots.KlinePatternBias {
	if out.KlinePatternBias != nil {
		return out.KlinePatternBias
	}
	if out.KlinePattern != nil && out.KlinePattern.Bias != nil {
		return out.KlinePattern.Bias
	}
	return nil
}

func klineBiasToDirection(b *slots.KlinePatternBias) LabJEVUnifiedDirection {
	if b == nil {
		return LabJEVUnifiedUnclear
	}
	switch strings.TrimSpace(b.Bias) {
	case "看多":
		return LabJEVUnifiedBull
	case "看空":
		return LabJEVUnifiedBear
	default:
		return LabJEVUnifiedUnclear
	}
}

func patternConfidenceFromBias(b *slots.KlinePatternBias, hits []slots.KlinePatternHit) float64 {
	bull, bear := countDirectionalPatternHits(hits)
	if bull+bear == 0 && b != nil {
		absScore := math.Abs(b.Score)
		if absScore == 0 {
			return 0
		}
		return clampUnitInterval(math.Min(1, absScore/3))
	}
	if bull+bear == 0 {
		return 0
	}
	major := math.Max(float64(bull), float64(bear))
	return clampUnitInterval(major / float64(bull+bear))
}

func countDirectionalPatternHits(hits []slots.KlinePatternHit) (bull, bear int) {
	for _, h := range hits {
		switch h.Direction {
		case 1:
			bull++
		case -1:
			bear++
		}
	}
	return bull, bear
}

func synthesizeLabJEVUnified(structure, pattern jevUnifiedLeg) *LabJEVUnified {
	out := &LabJEVUnified{
		Breakdown: LabJEVUnifiedBreakdown{
			Structure: structure.summary,
			Pattern:   pattern.summary,
		},
	}

	if !structure.present {
		out.Direction = LabJEVUnifiedUnclear
		out.Confidence = 0
		out.Conflict = true
		if out.Breakdown.Pattern == "" {
			out.Breakdown.Pattern = pattern.summary
		}
		return out
	}

	if !pattern.present {
		out.Direction = structure.dir
		out.Confidence = clampUnitInterval(structure.confidence)
		out.Conflict = false
		if structure.dir == LabJEVUnifiedUnclear {
			out.Conflict = true
		}
		return out
	}

	if structure.dir == LabJEVUnifiedUnclear || structure.dir != pattern.dir {
		out.Direction = LabJEVUnifiedUnclear
		out.Confidence = clampUnitInterval(0.7*structure.confidence + 0.3*pattern.confidence)
		out.Conflict = true
		return out
	}

	out.Direction = structure.dir
	out.Confidence = clampUnitInterval(0.7*structure.confidence + 0.3*pattern.confidence)
	out.Conflict = false
	return out
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
