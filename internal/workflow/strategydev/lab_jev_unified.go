package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const labJEVUnifiedConflictConfidenceCap = 0.75

const (
	labJEVLegNamePA     = "价格结构 JEV (price_analysis_jev)"
	labJEVLegNameKline  = "K 线形态 JEV (kline_jev)"
	labJEVLegNameTA     = "技术指标 JEV (technical_analysis_jev)"
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
	Technical string `json:"technical"`
}

// LabJEVUnified is the total JEV read synthesizing price / kline / TA JEV facts and conclusions.
type LabJEVUnified struct {
	Source            string                   `json:"source"`
	Model             string                   `json:"model"`
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	RecommendedReason string                   `json:"recommended_reason"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	OK                bool                     `json:"ok"`
	Error             string                   `json:"error"`
	Direction         LabJEVUnifiedDirection   `json:"direction"`
	Confidence        float64                  `json:"confidence"`
	Conflict          bool                     `json:"conflict"`
	Breakdown         LabJEVUnifiedBreakdown   `json:"breakdown"`
}

var labJEVUnifiedCriteria = map[string]string{
	"看多": "参与各路的基础事实与 JEV 结论整体偏多，或冲突时经权衡后仍偏多（非固定权重计票）",
	"看空": "参与各路的基础事实与 JEV 结论整体偏空，或冲突时经权衡后仍偏空（非固定权重计票）",
	"不明确": "参与路信息均不足，或经权衡仍无法可靠定向（非仅因冲突就默认此项）",
}

type labJEVLegSnapshot struct {
	name    string
	facts   string
	choice  string
	reason  string
	choices []PriceAnalysisJEVChoice
}

// BuildLabJEVUnifiedState feeds unified JEV Choice with per-leg fact summaries plus JEV conclusions; omitted legs are labeled.
func BuildLabJEVUnifiedState(plan labJEVUnifiedPlan, legs []labJEVLegSnapshot, signalSide string, longOnly bool) string {
	var b strings.Builder
	if heading := LabSignalSideTriggerHeading(signalSide, longOnly); heading != "" {
		b.WriteString(heading)
		b.WriteByte('\n')
	}
	legByName := map[string]labJEVLegSnapshot{}
	for _, leg := range legs {
		legByName[leg.name] = leg
	}
	appendLabJEVUnifiedLegSection(&b, labJEVLegNamePA, plan.attemptPA, legByName[labJEVLegNamePA])
	appendLabJEVUnifiedLegSection(&b, labJEVLegNameKline, plan.attemptKline, legByName[labJEVLegNameKline])
	appendLabJEVUnifiedLegSection(&b, labJEVLegNameTA, plan.attemptTA, legByName[labJEVLegNameTA])
	return strings.TrimSpace(b.String())
}

func appendLabJEVUnifiedLegSection(b *strings.Builder, title string, participated bool, leg labJEVLegSnapshot) {
	fmt.Fprintf(b, "\n--- %s ---\n", title)
	if !participated {
		b.WriteString("参与状态：未参与（本 episode 无该路输入或未启用）\n")
		return
	}
	if f := strings.TrimSpace(leg.facts); f != "" {
		b.WriteString("基础事实摘要：\n")
		b.WriteString(f)
		b.WriteByte('\n')
	}
	b.WriteString("JEV 结论：\n")
	fmt.Fprintf(b, "推荐：%s\n", strings.TrimSpace(leg.choice))
	if len(leg.choices) > 0 {
		b.WriteString("选项概率：")
		for i, c := range leg.choices {
			if i > 0 {
				b.WriteString(" ")
			}
			fmt.Fprintf(b, "%s=%.3f", c.Label, c.Probability)
		}
		b.WriteByte('\n')
	}
	if r := strings.TrimSpace(leg.reason); r != "" {
		fmt.Fprintf(b, "理由：%s\n", r)
	}
}

// ResolveLabJEVUnified calls JEV Choice on three-leg facts + conclusions; never mirrors a single leg locally.
func ResolveLabJEVUnified(
	ctx context.Context,
	jev *JevClient,
	out *LabEpisodeDetail,
	plan labJEVUnifiedPlan,
	signalSide, signalFreq string,
	longOnly bool,
) LabJEVUnified {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyLabJEVUnified(model)
	legs := collectLabJEVUnifiedLegs(out, plan, signalSide, signalFreq, longOnly)
	state := BuildLabJEVUnifiedState(plan, legs, signalSide, longOnly)
	if state == "" {
		base.Error = "lab unified decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中各路「基础事实摘要」与同路的 JEV 结论（推荐、概率、理由），综合判断该信号上下文最终方向。" +
		"未参与的路须忽略；不得仅用结论而忽略事实摘要；不得对三路做固定权重或机械计票合成方向，须按上下文权衡。" +
		"只能从给定选项中选一项。规则：①参与路同向则倾向该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅部分路有明确倾向时跟随有信息的一方；④均不足或确实无法判断时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, labJEVUnifiedCriteria)
	if err != nil {
		base.Error = err.Error()
		return base
	}
	if respModel != "" {
		base.Model = respModel
	}
	idx, ok := indexPriceAnalysisJEVLabel(choice)
	if !ok {
		base.Error = fmt.Sprintf("unexpected jev choice: %q", choice)
		return base
	}
	base.Choices = priceAnalysisJEVChoicesFromProbs(probs)
	base.RecommendedChoice = choice
	base.RecommendedIndex = idx
	conf := 0.0
	if idx >= 0 && idx < len(priceAnalysisJEVLabels) {
		if p, ok := probs[priceAnalysisJEVLabels[idx]]; ok {
			conf = clampUnitInterval(p)
		}
	}
	if confidence > 0 {
		conf = clampUnitInterval(confidence)
	}
	conflict := labJEVUnifiedLegConflict(legs, choice)
	if conflict && conf > labJEVUnifiedConflictConfidenceCap {
		conf = labJEVUnifiedConflictConfidenceCap
	}
	legReasonInputs := labJEVUnifiedLegReasonInputs(plan, legs)
	base.RecommendedReason = labJEVUnifiedReasonFromSummary(ctx, jev, choice, idx, base.Choices, conf, legReasonInputs)
	base.OK = true
	base.Error = ""
	base.Direction = paJEVLabelToDirection(choice)
	base.Confidence = conf
	base.Conflict = conflict
	base.Breakdown = labJEVUnifiedBreakdownFromLegs(legs)
	return base
}

func emptyLabJEVUnified(model string) LabJEVUnified {
	return LabJEVUnified{
		Source:  "jev",
		Model:   model,
		Choices: priceAnalysisJEVChoicesFromProbs(nil),
		OK:      false,
	}
}

func labJEVUnifiedBreakdownFromLegs(legs []labJEVLegSnapshot) LabJEVUnifiedBreakdown {
	var bd LabJEVUnifiedBreakdown
	for _, leg := range legs {
		line := legSummaryLine(leg)
		switch leg.name {
		case labJEVLegNamePA:
			bd.Structure = line
		case labJEVLegNameKline:
			bd.Pattern = line
		case labJEVLegNameTA:
			bd.Technical = line
		}
	}
	return bd
}

func labJEVUnifiedLegReasonInputs(plan labJEVUnifiedPlan, legs []labJEVLegSnapshot) []labJEVUnifiedLegReasonInput {
	legByName := map[string]labJEVLegSnapshot{}
	for _, leg := range legs {
		legByName[leg.name] = leg
	}
	type slot struct {
		name string
		on   bool
	}
	slots := []slot{
		{labJEVLegNamePA, plan.attemptPA},
		{labJEVLegNameKline, plan.attemptKline},
		{labJEVLegNameTA, plan.attemptTA},
	}
	var out []labJEVUnifiedLegReasonInput
	for _, s := range slots {
		in := labJEVUnifiedLegReasonInput{Name: s.name, Participated: s.on}
		if s.on {
			if leg, ok := legByName[s.name]; ok {
				in.RecommendedChoice = strings.TrimSpace(leg.choice)
				in.RecommendedReason = strings.TrimSpace(leg.reason)
				in.Facts = strings.TrimSpace(leg.facts)
			}
		}
		out = append(out, in)
	}
	return out
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

func labJEVLegFromPA(out *LabEpisodeDetail, signalSide string, longOnly bool) *labJEVLegSnapshot {
	j := out.PriceAnalysisJEV
	if j == nil || !j.OK {
		return nil
	}
	facts := ""
	if out.PriceAnalysis != nil {
		facts = BuildPriceAnalysisJEVState(out.PriceAnalysis, signalSide, longOnly)
	}
	return &labJEVLegSnapshot{
		name:    labJEVLegNamePA,
		facts:   facts,
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

func labJEVLegFromKline(out *LabEpisodeDetail, signalSide string, longOnly bool) *labJEVLegSnapshot {
	j := out.KlineJEV
	if j == nil || !j.OK {
		return nil
	}
	kline := labJEVKlineFromDetail(out)
	facts := BuildKlineJEVState(kline, signalSide, longOnly)
	return &labJEVLegSnapshot{
		name:    labJEVLegNameKline,
		facts:   facts,
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

func labJEVLegFromTA(out *LabEpisodeDetail, signalSide, signalFreq string, longOnly bool) *labJEVLegSnapshot {
	j := out.TechnicalAnalysisJEV
	if j == nil || !j.OK {
		return nil
	}
	facts := ""
	if out.TechnicalAnalysis != nil {
		facts = BuildTechnicalAnalysisJEVState(out.TechnicalAnalysis, signalSide, signalFreq, longOnly)
	}
	return &labJEVLegSnapshot{
		name:    labJEVLegNameTA,
		facts:   facts,
		choice:  j.RecommendedChoice,
		reason:  j.RecommendedReason,
		choices: j.Choices,
	}
}

type labJEVUnifiedPlan struct {
	attemptTA    bool
	attemptPA    bool
	attemptKline bool
}

func labJEVUnifiedPlanFor(in LabDetailInput, out *LabEpisodeDetail) labJEVUnifiedPlan {
	plan := labJEVUnifiedPlan{attemptTA: in.TAClient != nil}
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
	if plan.attemptTA {
		if out.TechnicalAnalysisJEV == nil || !out.TechnicalAnalysisJEV.OK {
			return true
		}
	}
	if !plan.attemptPA && !plan.attemptKline && !plan.attemptTA {
		return true
	}
	return false
}

func collectLabJEVUnifiedLegs(out *LabEpisodeDetail, plan labJEVUnifiedPlan, signalSide, signalFreq string, longOnly bool) []labJEVLegSnapshot {
	var legs []labJEVLegSnapshot
	if plan.attemptPA {
		if leg := labJEVLegFromPA(out, signalSide, longOnly); leg != nil {
			legs = append(legs, *leg)
		}
	}
	if plan.attemptKline {
		if leg := labJEVLegFromKline(out, signalSide, longOnly); leg != nil {
			legs = append(legs, *leg)
		}
	}
	if plan.attemptTA {
		if leg := labJEVLegFromTA(out, signalSide, signalFreq, longOnly); leg != nil {
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
	rec := ResolveLabJEVUnified(ctx, in.Jev, out, plan, side, in.SignalFreq, in.LongOnly)
	out.JEVUnified = &rec
}

// ApplyLabJEVUnified is kept for tests that inject leg snapshots without a live JEV client.
func ApplyLabJEVUnified(out *LabEpisodeDetail) {
	if out == nil {
		return
	}
	var legs []labJEVLegSnapshot
	if leg := labJEVLegFromPA(out, "", false); leg != nil {
		legs = append(legs, *leg)
	}
	if leg := labJEVLegFromKline(out, "", false); leg != nil {
		legs = append(legs, *leg)
	}
	if leg := labJEVLegFromTA(out, "", "", false); leg != nil {
		legs = append(legs, *leg)
	}
	if len(legs) == 0 {
		return
	}
	if len(legs) == 1 {
		leg := legs[0]
		conf := jevChoiceConfidenceFromLeg(leg)
		out.JEVUnified = &LabJEVUnified{
			OK:                true,
			Source:            "jev",
			RecommendedChoice: leg.choice,
			RecommendedIndex:  legChoiceIndex(leg),
			RecommendedReason: legSummaryLine(leg),
			Choices:           leg.choices,
			Direction:         paJEVLabelToDirection(leg.choice),
			Confidence:        conf,
			Conflict:          false,
			Breakdown:         labJEVUnifiedBreakdownFromLegs(legs),
		}
		out.JEVUnified.Conflict = labJEVDisplayConflict(out, leg.choice)
		if out.JEVUnified.Conflict && out.JEVUnified.Confidence > labJEVUnifiedConflictConfidenceCap {
			out.JEVUnified.Confidence = labJEVUnifiedConflictConfidenceCap
		}
	}
}

func legChoiceIndex(leg labJEVLegSnapshot) int {
	idx, ok := indexPriceAnalysisJEVLabel(leg.choice)
	if !ok {
		return -1
	}
	return idx
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

// labJEVDisplayConflict is informational for FE only (legacy single-leg mirror tests).
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
