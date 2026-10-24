package priceanalysis

import (
	"fmt"
	"math"
	"strings"
)

const defaultPriceAnalysisScoreThreshold = 0.0

// TFScoreBreakdown is one timeframe's subscore components for diagnose cards.
type TFScoreBreakdown struct {
	Subscore   float64 `json:"subscore"`
	Conclusion float64 `json:"conclusion,omitempty"`
	Position   float64 `json:"position,omitempty"`
	BosChoch   float64 `json:"bos_choch,omitempty"`
	Missing    bool    `json:"missing,omitempty"`
}

// MultiTFPriceScore is the weighted multi-timeframe score for episode gating.
type MultiTFPriceScore struct {
	Score       float64                       `json:"price_score"`
	TFScores    map[string]TFScoreBreakdown   `json:"tf_scores,omitempty"`
	ScoreReason string                        `json:"score_reason,omitempty"`
}

var multiTFScoreBaseWeights = map[string]float64{
	"weekly": 0.45,
	"daily":  0.35,
	"60m":    0.20,
}

// PriceAnalysisScoreThreshold returns the hard-gate threshold (default 0).
func PriceAnalysisScoreThreshold(threshold *float64) float64 {
	if threshold == nil {
		return defaultPriceAnalysisScoreThreshold
	}
	return *threshold
}

// ComputeMultiTFPriceScore scores 周线/日线/60m for lab composite (signal entry TF e.g. 5m is display-only).
func ComputeMultiTFPriceScore(pa PriceAnalysis, gateFreq string) MultiTFPriceScore {
	return computeWeightedMultiTFPriceScore(pa, compositeScoreWantSpecs(gateFreq), gateFreq)
}

// compositeScoreWantSpecs selects TFs for composite: fixed 60m+日线+周线 when signal is intraday;
// 日线 signal uses 日线+周线 only (weights renormalize; 60m is not fetched and not "missing").
func compositeScoreWantSpecs(gateFreq string) []struct {
	key   string
	label string
} {
	gate := NormalizeFrequency(gateFreq)
	if gate == "" {
		gate = "60m"
	}
	switch gateTFScoreKey(gate) {
	case "weekly":
		return []struct {
			key   string
			label string
		}{{"weekly", "周线"}}
	case "daily":
		return []struct {
			key   string
			label string
		}{
			{"weekly", "周线"},
			{"daily", "日线"},
		}
	default:
		return []struct {
			key   string
			label string
		}{
			{"weekly", "周线"},
			{"daily", "日线"},
			{"60m", "1小时"},
		}
	}
}

func compositeIncludes60m(gateFreq string) bool {
	for _, spec := range compositeScoreWantSpecs(gateFreq) {
		if spec.key == "60m" {
			return true
		}
	}
	return false
}

// ComputeHTFGatePriceScore is the buy-gate composite using only TFs coarser than gateFreq.
// Entry-period subscore is excluded; remaining base weights (0.45/0.35/0.20) renormalize over present HTFs.
// Weekly-only signal has no coarser TF — falls back to ComputeMultiTFPriceScore (gate equals display).
func ComputeHTFGatePriceScore(pa PriceAnalysis, gateFreq string) MultiTFPriceScore {
	want, ok := htfGateWantSpecs(gateFreq)
	if !ok || len(want) == 0 {
		return ComputeMultiTFPriceScore(pa, gateFreq)
	}
	return computeWeightedMultiTFPriceScore(pa, want, gateFreq)
}

func htfGateWantSpecs(gateFreq string) ([]struct {
	key   string
	label string
}, bool) {
	entryKey := gateTFScoreKey(gateFreq)
	candidates := []struct {
		key   string
		label string
	}{
		{"weekly", "周线"},
		{"daily", "日线"},
	}
	var want []struct {
		key   string
		label string
	}
	for _, spec := range candidates {
		if tfScoreRank(spec.key) > tfScoreRank(entryKey) {
			want = append(want, spec)
		}
	}
	if len(want) == 0 {
		return nil, false
	}
	return want, true
}

func tfScoreRank(key string) int {
	switch key {
	case "weekly":
		return 3
	case "daily":
		return 2
	case "60m":
		return 1
	case "5m":
		return 0
	default:
		return 1
	}
}

func computeWeightedMultiTFPriceScore(pa PriceAnalysis, want []struct {
	key   string
	label string
}, gateFreq string) MultiTFPriceScore {
	gate := NormalizeFrequency(gateFreq)
	if gate == "" {
		gate = "60m"
	}
	byLabel := map[string]PriceAnalysisRow{}
	for _, r := range pa.Rows {
		byLabel[r.Timeframe] = r
	}

	var missing []string
	var weighted float64
	var weightSum float64
	tfScores := map[string]TFScoreBreakdown{}
	var applied []compositeAppliedWeight

	for _, spec := range want {
		w := multiTFScoreBaseWeights[spec.key]
		if w <= 0 {
			continue
		}
		row, ok := byLabel[spec.label]
		if !ok || rowMissingForScore(row) {
			missing = append(missing, spec.label)
			tfScores[spec.label] = TFScoreBreakdown{Missing: true}
			continue
		}
		br := scorePriceAnalysisRow(row)
		tfScores[spec.label] = br
		weighted += w * br.Subscore
		weightSum += w
		applied = append(applied, compositeAppliedWeight{label: spec.label, w: w})
	}

	out := MultiTFPriceScore{TFScores: tfScores}
	if weightSum <= 0 {
		out.Score = 0
		if len(missing) > 0 {
			out.ScoreReason = formatScoreReason(0, missing)
		} else {
			out.ScoreReason = "多周期数据缺失，综合分 0"
		}
		return out
	}
	norm := weighted / weightSum
	out.Score = NormalizePriceScore(100 * clamp(norm/2, -1, 1))
	out.ScoreReason = formatScoreReason(out.Score, missing)
	if !compositeIncludes60m(gateFreq) && len(applied) > 0 {
		out.ScoreReason = appendCompositeWeightNote(out.ScoreReason, applied, weightSum)
	}
	return out
}

type compositeAppliedWeight struct {
	label string
	w     float64
}

// NormalizePriceScore rounds to 2 decimal places for gating/display and maps -0 to 0.
func NormalizePriceScore(score float64) float64 {
	rounded := math.Round(score*100) / 100
	if rounded == 0 {
		return 0
	}
	return rounded
}

func gateTFScoreKey(gate string) string {
	switch NormalizeFrequency(gate) {
	case "5m":
		return "5m"
	case "15m", "60m":
		return "60m"
	case "daily":
		return "daily"
	case "weekly", "1wk":
		return "weekly"
	default:
		return "60m"
	}
}

func rowMissingForScore(r PriceAnalysisRow) bool {
	if r.Structure == structurePriorTradingDayUnavailable || r.Structure == structureFetchFailed {
		return true
	}
	if r.Conclusion == RegimeUnclear && (r.Structure == structureInsufficientBars || r.Structure == structureNoBarsBeforeAnchor) {
		return true
	}
	return false
}

func scorePriceAnalysisRow(r PriceAnalysisRow) TFScoreBreakdown {
	br := TFScoreBreakdown{
		Conclusion: conclusionSubscore(r.Conclusion),
		Position:   positionSubscore(r.Position),
		BosChoch:   bosChochSubscore(r.BosChoch),
	}
	br.Subscore = br.Conclusion + br.Position + br.BosChoch
	br.Subscore = clamp(br.Subscore, -2, 2)
	return br
}

func conclusionSubscore(c string) float64 {
	switch strings.TrimSpace(c) {
	case RegimeUp:
		return 1.0
	case RegimeWeakUp:
		return 0.5
	case RegimeDown:
		return -1.0
	case RegimeWeakDown:
		return -0.5
	default:
		return 0
	}
}

func positionSubscore(p string) float64 {
	switch strings.TrimSpace(p) {
	case "靠近下沿":
		return 0.5
	case "靠近上沿":
		return -0.5
	default:
		return 0
	}
}

func bosChochSubscore(note string) float64 {
	n := strings.TrimSpace(note)
	switch {
	case n == "" || n == "无" || n == "-":
		return 0
	case strings.Contains(n, "CHOCH"):
		return -0.2
	case strings.Contains(n, "BOS") && strings.Contains(n, "放量"):
		return 0.3
	case strings.Contains(n, "BOS"):
		return 0.15
	default:
		return 0
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func appendCompositeWeightNote(reason string, applied []compositeAppliedWeight, weightSum float64) string {
	if weightSum <= 0 || len(applied) == 0 {
		return reason
	}
	var parts []string
	for _, a := range applied {
		pct := 100 * a.w / weightSum
		parts = append(parts, fmt.Sprintf("%s %.1f%%", a.label, pct))
	}
	return reason + fmt.Sprintf("；综合周期权重 %s", strings.Join(parts, "、"))
}

func formatScoreReason(score float64, missing []string) string {
	score = NormalizePriceScore(score)
	var tone string
	switch {
	case score >= 40:
		tone = "多周期偏多"
	case score > 0:
		tone = "略偏多"
	case score <= -40:
		tone = "多周期偏空"
	case score < 0:
		tone = "略偏空"
	default:
		tone = "中性"
	}
	if len(missing) > 0 {
		return fmt.Sprintf("%s（综合分 %.2f；缺周期：%s）", tone, score, strings.Join(missing, "、"))
	}
	return fmt.Sprintf("%s（综合分 %.2f）", tone, score)
}

// ShouldInterceptTriggerByScore applies hard gate from multi-TF score (not gate-TF regime alone).
func ShouldInterceptTriggerByScore(side string, score, threshold float64) bool {
	score = NormalizePriceScore(score)
	threshold = NormalizePriceScore(threshold)
	if side == "buy" {
		return score < threshold
	}
	return score > -threshold
}

func attachScoreToPriceAnalysis(pa *PriceAnalysis, gateFreq string) MultiTFPriceScore {
	if pa == nil {
		return MultiTFPriceScore{}
	}
	sc := ComputeMultiTFPriceScore(*pa, gateFreq)
	pa.PriceScore = sc.Score
	pa.TFScores = sc.TFScores
	pa.ScoreReason = sc.ScoreReason
	return sc
}
