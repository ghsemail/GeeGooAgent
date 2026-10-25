package klinepattern

import (
	"fmt"
	"math"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func neutralKlinePatternBias() slots.KlinePatternBias {
	return slots.KlinePatternBias{Score: 0, Bias: "中性", BiasColor: "gray"}
}

// biasFromThreeBarWindow scores all non-omitted bars in the three-bar window (§5.3).
func biasFromThreeBarWindow(win *slots.KlinePatternThreeBarWindow) slots.KlinePatternBias {
	if win == nil {
		return neutralKlinePatternBias()
	}
	if win.WindowBias != nil {
		return *win.WindowBias
	}
	var hits []slots.KlinePatternHit
	for _, bar := range win.Bars {
		if bar.Omitted {
			continue
		}
		hits = append(hits, bar.KlinePatterns...)
	}
	if len(hits) == 0 {
		return neutralKlinePatternBias()
	}
	return slots.ScoreKlinePatternHits(hits)
}

func roundKlinePatternScore(v float64) float64 {
	return math.Round(v*100) / 100
}

func biasFromWeightedScore(score float64) slots.KlinePatternBias {
	score = roundKlinePatternScore(score)
	b := neutralKlinePatternBias()
	b.Score = score
	if score > 0 {
		b.Bias = "看多"
		b.BiasColor = "green"
	} else if score < 0 {
		b.Bias = "看空"
		b.BiasColor = "red"
	}
	return b
}

// ComputeKlinePatternMTFWeightedConclusion aggregates ladder TF scores (§5.4).
// ok is true when any TF has pattern hits or |S_final| > 0.
func ComputeKlinePatternMTFWeightedConclusion(entries []slots.KlinePatternMTFEntry) (bias slots.KlinePatternBias, summary string, ok bool) {
	if len(entries) == 0 {
		return neutralKlinePatternBias(), "", false
	}
	weights := klinePatternMTFWeightsForLadder(len(entries))
	if len(weights) == 0 {
		return neutralKlinePatternBias(), "", false
	}
	var weighted float64
	anyHits := false
	nonZeroTF := 0
	var bullTF, bearTF []string
	for i := 0; i < len(entries) && i < len(weights); i++ {
		w := weights[i]
		e := entries[i]
		score := 0.0
		if e.Bias != nil {
			score = e.Bias.Score
		}
		weighted += w * score
		if threeBarHasPatternHits(e.ThreeBar) {
			anyHits = true
		}
		if score != 0 {
			nonZeroTF++
			label := strings.TrimSpace(e.Label)
			if label == "" {
				label = timeframeDisplayLabel(e.Frequency)
			}
			if score > 0 {
				bullTF = append(bullTF, label)
			} else if score < 0 {
				bearTF = append(bearTF, label)
			}
		}
	}
	bias = biasFromWeightedScore(weighted)
	if !anyHits && bias.Score == 0 {
		return bias, "", false
	}
	ok = true
	summary = klinePatternMTFWeightedSummary(bias, bullTF, bearTF, nonZeroTF)
	return bias, summary, ok
}

func threeBarHasPatternHits(win *slots.KlinePatternThreeBarWindow) bool {
	if win == nil {
		return false
	}
	for _, bar := range win.Bars {
		if !bar.Omitted && len(bar.KlinePatterns) > 0 {
			return true
		}
	}
	return false
}

func klinePatternMTFWeightedSummary(final slots.KlinePatternBias, bullTF, bearTF []string, nonZeroTF int) string {
	if final.Score == 0 && nonZeroTF == 0 {
		return "多周期形态：中性"
	}
	head := fmt.Sprintf("多周期加权%s（%.2f）", final.Bias, final.Score)
	if len(bullTF) > 0 && len(bearTF) > 0 {
		return head + "：周期分歧（" + strings.Join(bullTF, "、") + " 偏多；" + strings.Join(bearTF, "、") + " 偏空）"
	}
	if len(bullTF) > 0 {
		return head + "：" + strings.Join(bullTF, "、") + " 贡献偏多"
	}
	if len(bearTF) > 0 {
		return head + "：" + strings.Join(bearTF, "、") + " 贡献偏空"
	}
	return head
}

