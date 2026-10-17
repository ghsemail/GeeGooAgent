package strategydev

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// PreserveProbeLookbackBars snapshots full-span bars before months_back clamp (for per-signal lookback).
func PreserveProbeLookbackBars(data map[string]any) {
	if data == nil {
		return
	}
	if _, ok := data["bars_lookback"]; ok {
		return
	}
	bars, _ := data["bars"].([]any)
	if len(bars) == 0 {
		return
	}
	data["bars_lookback"] = append([]any(nil), bars...)
}

// ProbeLookbackBarsFromData returns pre-clamp bars when present, else current bars.
func ProbeLookbackBarsFromData(data map[string]any) []any {
	if data == nil {
		return nil
	}
	if lb, ok := data["bars_lookback"].([]any); ok && len(lb) > 0 {
		return lb
	}
	bars, _ := data["bars"].([]any)
	return bars
}

// ClampProbeSeriesToMonthsBack keeps only bars in [windowEnd−monthsBack calendar months, windowEnd]
// and slices per-bar aligned arrays (merged signals, rules, key_levels_series, etc.).
func ClampProbeSeriesToMonthsBack(data map[string]any, months int) {
	if data == nil {
		return
	}
	PreserveProbeLookbackBars(data)
	months = CapMonthsBack(months)
	bars, _ := data["bars"].([]any)
	if len(bars) == 0 {
		return
	}
	windowEnd := diagnoseWindowEnd(data, bars)
	if windowEnd.IsZero() {
		return
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	windowStart := windowEnd.In(loc).AddDate(0, -months, 0)

	startIdx, endIdx := barIndexRangeForWindow(bars, windowStart, windowEnd)
	if startIdx == 0 && endIdx == len(bars) {
		return
	}
	if endIdx <= startIdx {
		data["bars"] = []any{}
		data["buy_merged"] = []any{}
		data["sell_merged"] = []any{}
		data["bar_count"] = 0
		data["buy_hits"] = 0
		data["sell_hits"] = 0
		return
	}

	oldLen := len(bars)
	data["bars"] = bars[startIdx:endIdx]
	if merged, ok := data["buy_merged"].([]any); ok && len(merged) == oldLen {
		data["buy_merged"] = merged[startIdx:endIdx]
	}
	if merged, ok := data["sell_merged"].([]any); ok && len(merged) == oldLen {
		data["sell_merged"] = merged[startIdx:endIdx]
	}
	if hits, ok := data["pattern_hits"].([]any); ok && len(hits) == oldLen {
		data["pattern_hits"] = hits[startIdx:endIdx]
	}
	if rules, ok := data["buy_rules"].([]any); ok {
		data["buy_rules"] = sliceSignalRules(rules, startIdx, endIdx, oldLen)
	}
	if rules, ok := data["sell_rules"].([]any); ok {
		data["sell_rules"] = sliceSignalRules(rules, startIdx, endIdx, oldLen)
	}
	if kl, ok := data["key_levels_series"].(map[string]any); ok {
		data["key_levels_series"] = sliceKeyLevelSeries(kl, startIdx, endIdx, oldLen)
	}
	data["bar_count"] = endIdx - startIdx
	if buyM, ok := data["buy_merged"].([]any); ok {
		data["buy_hits"] = countMergedSignalHits(buyM, 1)
	}
	if sellM, ok := data["sell_merged"].([]any); ok {
		data["sell_hits"] = countMergedSignalHits(sellM, -1)
	}
}

func diagnoseWindowEnd(data map[string]any, bars []any) time.Time {
	if data != nil {
		for _, key := range []string{"as_of", "asOf", "end", "window_end"} {
			if v := data[key]; v != nil {
				if ts := parseDiagnoseTimeString(strings.TrimSpace(strings.TrimSpace(fmt.Sprint(v)))); !ts.IsZero() {
					return ts
				}
			}
		}
	}
	_, latest := barSeriesTimeExtent(bars)
	return latest
}

func parseDiagnoseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	row := map[string]any{"time": s}
	return barTimeUnix(row)
}

func barIndexRangeForWindow(bars []any, windowStart, windowEnd time.Time) (startIdx, endIdx int) {
	endIdx = len(bars)
	for i, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() {
			continue
		}
		if ts.After(windowEnd) {
			endIdx = i
			break
		}
	}
	for i := 0; i < endIdx; i++ {
		row, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() {
			continue
		}
		if !ts.Before(windowStart) {
			startIdx = i
			break
		}
	}
	return startIdx, endIdx
}

func sliceSignalRules(rules []any, startIdx, endIdx, barLen int) []any {
	out := make([]any, 0, len(rules))
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		cp := map[string]any{}
		for k, v := range rule {
			cp[k] = v
		}
		if ser, ok := rule["signal_series"].([]any); ok && len(ser) == barLen {
			cp["signal_series"] = ser[startIdx:endIdx]
		}
		if vs, ok := rule["value_series"].(map[string]any); ok {
			cp["value_series"] = sliceValueSeriesMap(vs, startIdx, endIdx, barLen)
		}
		out = append(out, cp)
	}
	return out
}

func sliceValueSeriesMap(vs map[string]any, startIdx, endIdx, barLen int) map[string]any {
	out := make(map[string]any, len(vs))
	for k, raw := range vs {
		arr, ok := raw.([]any)
		if !ok {
			out[k] = raw
			continue
		}
		out[k] = slicePerBarAnyArray(arr, startIdx, endIdx, barLen)
	}
	return out
}

func slicePerBarAnyArray(arr []any, startIdx, endIdx, barLen int) []any {
	if len(arr) == barLen {
		return arr[startIdx:endIdx]
	}
	if len(arr) > barLen {
		off := len(arr) - barLen
		return arr[off+startIdx : off+endIdx]
	}
	if startIdx >= len(arr) {
		return []any{}
	}
	if endIdx > len(arr) {
		endIdx = len(arr)
	}
	return arr[startIdx:endIdx]
}

func sliceKeyLevelSeries(kl map[string]any, startIdx, endIdx, barLen int) map[string]any {
	out := make(map[string]any, len(kl))
	for k, raw := range kl {
		switch t := raw.(type) {
		case []any:
			out[k] = slicePerBarAnyArray(t, startIdx, endIdx, barLen)
		case []float64:
			sliced := slicePerBarFloatArray(t, startIdx, endIdx, barLen)
			anySlice := make([]any, len(sliced))
			for i, v := range sliced {
				anySlice[i] = v
			}
			out[k] = anySlice
		default:
			out[k] = raw
		}
	}
	return out
}

func slicePerBarFloatArray(arr []float64, startIdx, endIdx, barLen int) []float64 {
	if len(arr) == barLen {
		return arr[startIdx:endIdx]
	}
	if len(arr) > barLen {
		off := len(arr) - barLen
		return arr[off+startIdx : off+endIdx]
	}
	if startIdx >= len(arr) {
		return nil
	}
	if endIdx > len(arr) {
		endIdx = len(arr)
	}
	return arr[startIdx:endIdx]
}

func countMergedSignalHits(merged []any, target int) int {
	n := 0
	for _, raw := range merged {
		if int(slots.FloatAny(raw)) == target {
			n++
		}
	}
	return n
}
