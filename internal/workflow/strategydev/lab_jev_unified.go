package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabJEVUnifiedDirection is the final lab-detail direction from the unified JEV 决策 pass.
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

// LabJEVUnified is the total JEV read synthesizing price / kline / TA JEV conclusions only.
type LabJEVUnified struct {
	Direction  LabJEVUnifiedDirection `json:"direction"`
	Confidence float64                `json:"confidence"`
	Conflict   bool                   `json:"conflict"`
	Breakdown  LabJEVUnifiedBreakdown `json:"breakdown"`
}

var labJEVUnifiedCriteria = map[string]string{
	"看多": "三路 JEV 结论整体偏多，或冲突时权衡后仍偏多",
	"看空": "三路 JEV 结论整体偏空，或冲突时权衡后仍偏空",
	"不明确": "三路结论均不足，或经权衡仍无法可靠定向（非仅因冲突就默认此项）",
}

type labJEVLegSnapshot struct {
	name   string
	choice string
	reason string
	choices []PriceAnalysisJEVChoice
}

// BuildLabJEVUnifiedState compresses three JEV legs (choice / probs / reason only) for the final pass.
func BuildLabJEVUnifiedState(legs []labJEVLegSnapshot, signalSide string) string {
	if len(legs) == 0 {
		return ""
	}
	var b strings.Builder
	side := strings.TrimSpace(signalSide)
	if side == "buy" {
		b.WriteString("信号侧：买入触发\n")
	} else if side == "sell" {
		b.WriteString("信号侧：卖出触发\n")
	}
	for _, leg := range legs {
		fmt.Fprintf(&b, "\n--- %s ---\n", leg.name)
		fmt.Fprintf(&b, "推荐：%s\n", strings.TrimSpace(leg.choice))
		if len(leg.choices) > 0 {
			b.WriteString("选项概率：")
			for i, c := range leg.choices {
				if i > 0 {
					b.WriteString(" ")
				}
				fmt.Fprintf(&b, "%s=%.3f", c.Label, c.Probability)
			}
			b.WriteByte('\n')
		}
		if r := strings.TrimSpace(leg.reason); r != "" {
			fmt.Fprintf(&b, "理由：%s\n", r)
		}
	}
	return strings.TrimSpace(b.String())
}

// ResolveLabJEVUnified calls JEV Choice on three leg conclusions; never mirrors a single leg locally.
func ResolveLabJEVUnified(ctx context.Context, jev *JevClient, legs []labJEVLegSnapshot, signalSide string) (LabJEVUnified, error) {
	var empty LabJEVUnified
	state := BuildLabJEVUnifiedState(legs, signalSide)
	if state == "" {
		return empty, fmt.Errorf("lab unified decision state empty")
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		return empty, fmt.Errorf("jev not configured")
	}
	instructions := "基于 state 中已给出的三路 JEV 结论（价格结构 / K 线形态 / 技术指标），综合判断该信号上下文最终方向。" +
		"不得重新推断原始行情，只能权衡三路 JEV 的推荐、概率与理由。只能从给定选项中选一项。" +
		"规则：①三路同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅一路或两路有明确倾向时跟随有信息的一方；④均不足或确实无法判断时才选不明确。"
	choice, probs, _, _, err := jev.Choice(ctx, state, "direction", instructions, labJEVUnifiedCriteria)
	if err != nil {
		return empty, err
	}
	idx, ok := indexPriceAnalysisJEVLabel(choice)
	if !ok {
		return empty, fmt.Errorf("unexpected jev choice: %q", choice)
	}
	conf := 0.0
	if idx >= 0 && idx < len(priceAnalysisJEVLabels) {
		if p, ok := probs[priceAnalysisJEVLabels[idx]]; ok {
			conf = clampUnitInterval(p)
		}
	}
	out := LabJEVUnified{
		Direction:  paJEVLabelToDirection(choice),
		Confidence: conf,
		Conflict:   labJEVUnifiedLegConflict(legs, choice),
	}
	for _, leg := range legs {
		switch leg.name {
		case "价格结构 JEV (price_analysis_jev)":
			out.Breakdown.Structure = legSummaryLine(leg)
		case "K 线形态 JEV (kline_jev)":
			out.Breakdown.Pattern = legSummaryLine(leg)
		}
	}
	if out.Breakdown.Structure == "" && len(legs) > 0 {
		out.Breakdown.Structure = legSummaryLine(legs[0])
	}
	return out, nil
}

func legSummaryLine(leg labJEVLegSnapshot) string {
	r := strings.TrimSpace(leg.reason)
	if r != "" {
		return r
	}
	if c := strings.TrimSpace(leg.choice); c != "" {
		return "JEV 决策：" + c
	}
	return ""
}

func labJEVUnifiedLegConflict(legs []labJEVLegSnapshot, unifiedChoice string) bool {
	unifiedChoice = strings.TrimSpace(unifiedChoice)
	if unifiedChoice == "" || unifiedChoice == "不明确" {
		return false
	}
	for _, leg := range legs {
		c := strings.TrimSpace(leg.choice)
		if c == "" || c == "不明确" {
			continue
		}
		if c != unifiedChoice {
			return true
		}
	}
	// Also flag when legs disagree with each other.
	var first string
	for _, leg := range legs {
		c := strings.TrimSpace(leg.choice)
		if c == "" || c == "不明确" {
			continue
		}
		if first == "" {
			first = c
			continue
		}
		if c != first {
			return true
		}
	}
	return false
}

func labJEVLegFromPA(j *PriceAnalysisJEV) *labJEVLegSnapshot {
	if j == nil || !j.OK {
		return nil
	}
	return &labJEVLegSnapshot{
		name:    "价格结构 JEV (price_analysis_jev)",
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

func labJEVLegFromKline(j *KlineJEV) *labJEVLegSnapshot {
	if j == nil || !j.OK {
		return nil
	}
	return &labJEVLegSnapshot{
		name:    "K 线形态 JEV (kline_jev)",
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

func labJEVLegFromTA(j *TechnicalAnalysisJEV) *labJEVLegSnapshot {
	if j == nil || !j.OK {
		return nil
	}
	return &labJEVLegSnapshot{
		name:    "技术指标 JEV (technical_analysis_jev)",
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

type labJEVUnifiedPlan struct {
	requireTA bool
	attemptPA bool
	attemptKline bool
}

func labJEVUnifiedPlanFor(in LabDetailInput, out *LabEpisodeDetail) labJEVUnifiedPlan {
	plan := labJEVUnifiedPlan{requireTA: in.TAClient != nil}
	if out.PriceAnalysis != nil && labPAInformative(out.PriceAnalysis) {
		plan.attemptPA = true
	}
	if kline := labJEVKlineFromDetail(out); !labJEVKlineInputEmpty(kline) {
		plan.attemptKline = true
	}
	return plan
}

func labJEVUnifiedBlocked(out *LabEpisodeDetail, plan labJEVUnifiedPlan) bool {
	if plan.attemptPA {
		if out.PriceAnalysisJEV == nil || !out.PriceAnalysisJEV.OK {
			return true
		}
	}
	if plan.attemptKline {
		if out.KlineJEV == nil || !out.KlineJEV.OK {
			return true
		}
	}
	if plan.requireTA {
		if out.TechnicalAnalysisJEV == nil || !out.TechnicalAnalysisJEV.OK {
			return true
		}
	}
	if !plan.attemptPA && !plan.attemptKline && !plan.requireTA {
		return true
	}
	return false
}

func collectLabJEVUnifiedLegs(out *LabEpisodeDetail, plan labJEVUnifiedPlan) []labJEVLegSnapshot {
	var legs []labJEVLegSnapshot
	if plan.attemptPA {
		if leg := labJEVLegFromPA(out.PriceAnalysisJEV); leg != nil {
			legs = append(legs, *leg)
		}
	}
	if plan.attemptKline {
		if leg := labJEVLegFromKline(out.KlineJEV); leg != nil {
			legs = append(legs, *leg)
		}
	}
	if plan.requireTA {
		if leg := labJEVLegFromTA(out.TechnicalAnalysisJEV); leg != nil {
			legs = append(legs, *leg)
		}
	}
	return legs
}

// applyLabJEVUnified runs the final JEV pass when all attempted legs succeeded.
func applyLabJEVUnified(ctx context.Context, out *LabEpisodeDetail, side string, in LabDetailInput) {
	if out == nil {
		return
	}
	plan := labJEVUnifiedPlanFor(in, out)
	if labJEVUnifiedBlocked(out, plan) {
		return
	}
	legs := collectLabJEVUnifiedLegs(out, plan)
	if len(legs) == 0 {
		return
	}
	unified, err := ResolveLabJEVUnified(ctx, in.Jev, legs, side)
	if err != nil {
		return
	}
	out.JEVUnified = &unified
}


// ApplyLabJEVUnified is kept for tests that inject leg snapshots without a live JEV client.
func ApplyLabJEVUnified(out *LabEpisodeDetail) {
	if out == nil {
		return
	}
	var legs []labJEVLegSnapshot
	if leg := labJEVLegFromPA(out.PriceAnalysisJEV); leg != nil {
		legs = append(legs, *leg)
	}
	if leg := labJEVLegFromKline(out.KlineJEV); leg != nil {
		legs = append(legs, *leg)
	}
	if leg := labJEVLegFromTA(out.TechnicalAnalysisJEV); leg != nil {
		legs = append(legs, *leg)
	}
	if len(legs) == 0 {
		return
	}
	// Local mirror only when all present legs are ok and exactly one leg (legacy tests).
	if len(legs) == 1 {
		leg := legs[0]
		out.JEVUnified = &LabJEVUnified{
			Direction:  paJEVLabelToDirection(leg.choice),
			Confidence: jevChoiceConfidenceFromLeg(leg),
			Conflict:   false,
			Breakdown: LabJEVUnifiedBreakdown{
				Structure: legSummaryLine(leg),
				Pattern:   patternSummaryText(out),
			},
		}
		if leg.name == "K 线形态 JEV (kline_jev)" {
			out.JEVUnified.Breakdown.Pattern = legSummaryLine(leg)
			out.JEVUnified.Breakdown.Structure = ""
		}
		out.JEVUnified.Conflict = labJEVDisplayConflict(out, leg.choice)
	}
}

func jevChoiceConfidenceFromLeg(leg labJEVLegSnapshot) float64 {
	idx, ok := indexPriceAnalysisJEVLabel(leg.choice)
	if !ok || idx < 0 || idx >= len(leg.choices) {
		return 0
	}
	return clampUnitInterval(leg.choices[idx].Probability)
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

// labJEVDisplayConflict is informational for FE only.
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
