package strategydev

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

// ComputeMultiTFPriceScore scores 周线/日线/gate-TF rows for buy-direction bias.
func ComputeMultiTFPriceScore(pa PriceAnalysis, gateFreq string) MultiTFPriceScore {
	gate := NormalizeFrequency(gateFreq)
	if gate == "" {
		gate = "60m"
	}
	want := []struct {
		key   string
		label string
	}{
		{"weekly", "周线"},
		{"daily", "日线"},
	}
	lowKey := gateTFScoreKey(gate)
	lowLabel := timeframeDisplayLabel(gate)
	if lowKey != "weekly" && lowKey != "daily" {
		want = append(want, struct {
			key   string
			label string
		}{lowKey, lowLabel})
	}
	byLabel := map[string]PriceAnalysisRow{}
	for _, r := range pa.Rows {
		byLabel[r.Timeframe] = r
	}

	var missing []string
	var weighted float64
	var weightSum float64
	tfScores := map[string]TFScoreBreakdown{}

	for _, spec := range want {
		w := multiTFScoreBaseWeights[spec.key]
		if w <= 0 {
			if spec.key != gateTFScoreKey(gate) {
				continue
			}
			w = 0.20
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
	return out
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
