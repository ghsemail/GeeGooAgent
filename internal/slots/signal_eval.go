package slots

import (
	"fmt"
	"math"
	"strings"
)

// SignalEpisodeDetail is one buy/sell holding period until the next opposite signal.
type SignalEpisodeDetail struct {
	Side              string  `json:"side"`
	StartIdx          int     `json:"start_idx"`
	EndIdx            int     `json:"end_idx"`
	OppositeIdx       int     `json:"opposite_idx,omitempty"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	OppositeTime      string  `json:"opposite_time,omitempty"`
	StartClose        float64 `json:"start_close"`
	EndClose          float64 `json:"end_close"`
	PriceReturn       float64 `json:"price_return"`
	DirectionReturn   float64 `json:"direction_return"`
	Hit               bool    `json:"hit"`
	HoldingBars       int     `json:"holding_bars"`
	Complete          bool    `json:"complete"`
}

// SignalEpisodeMetrics summarizes directional accuracy until the next opposite signal.
type SignalEpisodeMetrics struct {
	CompleteCount   int     `json:"complete_count"`
	IncompleteCount int     `json:"incomplete_count"`
	HitCount        int     `json:"hit_count"`
	HitRate         float64 `json:"hit_rate"`
	AvgReturn       float64 `json:"avg_return"`
	MedianReturn    float64 `json:"median_return"`
	AvgHoldingBars  float64 `json:"avg_holding_bars"`
	MedianHoldingBars float64 `json:"median_holding_bars"`
}

// SignalEpisodeEval is the result of episode-based signal evaluation.
type SignalEpisodeEval struct {
	Method       string                `json:"method"`
	BuyEpisodes  SignalEpisodeMetrics  `json:"buy_episodes"`
	SellEpisodes SignalEpisodeMetrics  `json:"sell_episodes"`
	BuyDetails   []SignalEpisodeDetail `json:"buy_details"`
	SellDetails  []SignalEpisodeDetail `json:"sell_details"`
}

const signalEvalMethodUntilOpposite = "until_opposite_signal"

// EvaluateSignalEpisodes scores buy/sell triggers using close-to-close return until the next opposite signal.
func EvaluateSignalEpisodes(bars []any, buyMerged, sellMerged []any) SignalEpisodeEval {
	closes := barCloses(bars)
	times := barTimes(bars)
	buyStarts := episodeStarts(buyMerged, 1)
	sellStarts := episodeStarts(sellMerged, -1)
	buyDetails := collectEpisodes("buy", buyStarts, closes, times, oppositeIndices(sellMerged, -1), true)
	sellDetails := collectEpisodes("sell", sellStarts, closes, times, oppositeIndices(buyMerged, 1), false)
	return SignalEpisodeEval{
		Method:       signalEvalMethodUntilOpposite,
		BuyDetails:   buyDetails,
		SellDetails:  sellDetails,
		BuyEpisodes:  metricsFromEpisodes(buyDetails),
		SellEpisodes: metricsFromEpisodes(sellDetails),
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

func barTimes(bars []any) []string {
	out := make([]string, len(bars))
	for i, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out[i] = strings.TrimSpace(fmt.Sprint(row["time"]))
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

func collectEpisodes(side string, starts []int, closes []float64, times []string, opposites []int, buySide bool) []SignalEpisodeDetail {
	if len(closes) == 0 {
		return nil
	}
	out := make([]SignalEpisodeDetail, 0, len(starts))
	for _, start := range starts {
		if start < 0 || start >= len(closes) || closes[start] <= 0 {
			continue
		}
		ep := SignalEpisodeDetail{
			Side:       side,
			StartIdx:   start,
			StartTime:  times[start],
			StartClose: closes[start],
		}
		end := -1
		oppIdx := -1
		for _, opp := range opposites {
			if opp <= start {
				continue
			}
			oppIdx = opp
			end = opp - 1
			break
		}
		if end < start {
			ep.Complete = false
			out = append(out, ep)
			continue
		}
		ep.Complete = true
		ep.EndIdx = end
		ep.EndTime = times[end]
		ep.EndClose = closes[end]
		ep.HoldingBars = end - start
		if oppIdx >= 0 && oppIdx < len(times) {
			ep.OppositeIdx = oppIdx
			ep.OppositeTime = times[oppIdx]
		}
		ep.PriceReturn = closes[end]/closes[start] - 1
		if buySide {
			ep.DirectionReturn = ep.PriceReturn
			ep.Hit = ep.PriceReturn > 0
		} else {
			ep.DirectionReturn = -ep.PriceReturn
			ep.Hit = ep.PriceReturn < 0
		}
		out = append(out, ep)
	}
	return out
}

func metricsFromEpisodes(details []SignalEpisodeDetail) SignalEpisodeMetrics {
	m := SignalEpisodeMetrics{}
	returns := make([]float64, 0)
	holds := make([]float64, 0)
	for _, ep := range details {
		if !ep.Complete {
			m.IncompleteCount++
			continue
		}
		returns = append(returns, ep.DirectionReturn)
		holds = append(holds, float64(ep.HoldingBars))
		if ep.Hit {
			m.HitCount++
		}
	}
	if len(returns) == 0 {
		return m
	}
	m.CompleteCount = len(returns)
	m.HitRate = float64(m.HitCount) / float64(len(returns))
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	m.AvgReturn = sum / float64(len(returns))
	m.MedianReturn = median(append([]float64(nil), returns...))
	m.AvgHoldingBars = sumFloats(holds) / float64(len(holds))
	m.MedianHoldingBars = median(append([]float64(nil), holds...))
	return m
}

func sumFloats(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s
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
	fmt.Fprintf(b, "%s_complete=%d %s_incomplete=%d %s_hit_rate=%.1f%% %s_avg_return=%.2f%% %s_median_return=%.2f%% %s_avg_hold_bars=%.1f %s_median_hold_bars=%.0f\n",
		side, m.CompleteCount, side, m.IncompleteCount, side, m.HitRate*100, side, m.AvgReturn*100, side, m.MedianReturn*100,
		side, m.AvgHoldingBars, side, m.MedianHoldingBars)
}

// FormatPct formats a fractional return for display.
func FormatPct(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	return fmt.Sprintf("%+.2f%%", v*100)
}
