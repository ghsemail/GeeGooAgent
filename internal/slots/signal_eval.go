package slots

import (
	"fmt"
	"math"
	"strings"
)

// SignalEpisodeMetrics summarizes directional accuracy until the next opposite signal.
type SignalEpisodeMetrics struct {
	CompleteCount   int     `json:"complete_count"`
	IncompleteCount int     `json:"incomplete_count"`
	HitCount        int     `json:"hit_count"`
	HitRate         float64 `json:"hit_rate"`
	AvgReturn       float64 `json:"avg_return"`
	MedianReturn    float64 `json:"median_return"`
}

// SignalEpisodeEval is the result of episode-based signal evaluation.
type SignalEpisodeEval struct {
	Method      string               `json:"method"`
	BuyEpisodes SignalEpisodeMetrics `json:"buy_episodes"`
	SellEpisodes SignalEpisodeMetrics `json:"sell_episodes"`
}

const signalEvalMethodUntilOpposite = "until_opposite_signal"

// EvaluateSignalEpisodes scores buy/sell triggers using close-to-close return until the next opposite signal.
func EvaluateSignalEpisodes(bars []any, buyMerged, sellMerged []any) SignalEpisodeEval {
	closes := barCloses(bars)
	buyStarts := episodeStarts(buyMerged, 1)
	sellStarts := episodeStarts(sellMerged, -1)
	buyReturns, buyIncomplete := collectEpisodeReturns(buyStarts, closes, oppositeIndices(sellMerged, -1), true)
	sellReturns, sellIncomplete := collectEpisodeReturns(sellStarts, closes, oppositeIndices(buyMerged, 1), false)
	return SignalEpisodeEval{
		Method:       signalEvalMethodUntilOpposite,
		BuyEpisodes:  metricsFromReturns(buyReturns, buyIncomplete),
		SellEpisodes: metricsFromReturns(sellReturns, sellIncomplete),
	}
}

func barCloses(bars []any) []float64 {
	out := make([]float64, len(bars))
	for i, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out[i] = floatAny(row["close"])
	}
	return out
}

func episodeStarts(merged []any, target int) []int {
	out := make([]int, 0)
	for i, raw := range merged {
		if mergedSignalValue(raw) != target {
			continue
		}
		if i > 0 && mergedSignalValue(merged[i-1]) == target {
			continue
		}
		out = append(out, i)
	}
	return out
}

func oppositeIndices(merged []any, target int) []int {
	out := make([]int, 0)
	for i, raw := range merged {
		if mergedSignalValue(raw) == target {
			out = append(out, i)
		}
	}
	return out
}

func collectEpisodeReturns(starts []int, closes []float64, opposites []int, buySide bool) (returns []float64, incomplete int) {
	if len(closes) == 0 {
		return nil, 0
	}
	for _, start := range starts {
		if start < 0 || start >= len(closes) || closes[start] <= 0 {
			continue
		}
		end := -1
		for _, opp := range opposites {
			if opp <= start {
				continue
			}
			end = opp - 1
			break
		}
		if end < start {
			incomplete++
			continue
		}
		ret := closes[end]/closes[start] - 1
		if !buySide {
			ret = -ret
		}
		returns = append(returns, ret)
	}
	return returns, incomplete
}

func metricsFromReturns(returns []float64, incomplete int) SignalEpisodeMetrics {
	m := SignalEpisodeMetrics{IncompleteCount: incomplete}
	if len(returns) == 0 {
		return m
	}
	m.CompleteCount = len(returns)
	hits := 0
	sum := 0.0
	sorted := append([]float64(nil), returns...)
	for _, r := range returns {
		sum += r
		if r > 0 {
			hits++
		}
	}
	m.HitCount = hits
	m.HitRate = float64(hits) / float64(len(returns))
	m.AvgReturn = sum / float64(len(returns))
	m.MedianReturn = median(sorted)
	return m
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func mergedSignalValue(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func floatAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		var f float64
		fmt.Sscan(t, &f)
		return f
	default:
		return 0
	}
}

// FormatEvalMetricsJSON renders evaluation metrics for LLM prompts.
func FormatEvalMetricsJSON(eval SignalEpisodeEval) string {
	var b strings.Builder
	fmt.Fprintf(&b, "method=%s\n", eval.Method)
	writeSideMetrics(&b, "buy", eval.BuyEpisodes)
	writeSideMetrics(&b, "sell", eval.SellEpisodes)
	return strings.TrimSpace(b.String())
}

func writeSideMetrics(b *strings.Builder, side string, m SignalEpisodeMetrics) {
	fmt.Fprintf(b, "%s_complete=%d %s_incomplete=%d %s_hit_rate=%.1f%% %s_avg_return=%.2f%% %s_median_return=%.2f%%\n",
		side, m.CompleteCount, side, m.IncompleteCount, side, m.HitRate*100, side, m.AvgReturn*100, side, m.MedianReturn*100)
}

// FormatPct formats a fractional return for display.
func FormatPct(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	return fmt.Sprintf("%+.2f%%", v*100)
}
