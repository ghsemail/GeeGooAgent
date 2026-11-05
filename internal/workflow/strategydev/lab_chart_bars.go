package strategydev

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabDetailChartBars returns diagnosis chart bars (post months_back clamp).
// detail_ref / start_idx / chart_probe.bars indices refer to this series, never bars_lookback.
func LabDetailChartBars(probeRaw map[string]any) []any {
	if probeRaw == nil {
		return nil
	}
	bars, _ := probeRaw["bars"].([]any)
	return bars
}

// EnsureProbeChartBarsForLab clamps probe bars in place when bars still span the full
// lookback window (common on early probe persist before runner finalize).
func EnsureProbeChartBarsForLab(probeRaw map[string]any, months int) {
	if probeRaw == nil {
		return
	}
	months = CapMonthsBack(months)
	if months <= 0 {
		months = 3
	}
	chart, _ := probeRaw["bars"].([]any)
	lookback, _ := probeRaw["bars_lookback"].([]any)
	if len(chart) == 0 || len(lookback) == 0 {
		return
	}
	if len(chart) == len(lookback) {
		FinalizeDiagnoseProbeSeries(probeRaw, months, nil, nil)
	}
}

func signalEvalEpisodeByStartIdx(eval slots.SignalEpisodeEval, side string, startIdx int) (slots.SignalEpisodeDetail, bool) {
	var list []slots.SignalEpisodeDetail
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		list = eval.BuyDetails
	case "sell":
		list = eval.SellDetails
	default:
		return slots.SignalEpisodeDetail{}, false
	}
	for _, d := range list {
		if d.StartIdx == startIdx {
			return d, true
		}
	}
	return slots.SignalEpisodeDetail{}, false
}

func barTimeString(row map[string]any) string {
	if row == nil {
		return ""
	}
	return strings.TrimSpace(formatBarTimeAsiaShanghai(row))
}

func barTimeStringsEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// resolveLabDetailChartBar picks the chart bar row for detail_ref suffix idx.
// signal_eval start_time wins when bars[idx] is from a mismatched (lookback) coordinate system.
func resolveLabDetailChartBar(chartBars []any, eval slots.SignalEpisodeEval, side string, idx int) (chartIdx int, row map[string]any, ok bool) {
	if len(chartBars) == 0 || idx < 0 {
		return 0, nil, false
	}
	expectedTime := ""
	if ep, found := signalEvalEpisodeByStartIdx(eval, side, idx); found {
		expectedTime = strings.TrimSpace(ep.StartTime)
	}
	if idx < len(chartBars) {
		row = barRowAt(chartBars, idx)
		if expectedTime == "" || barTimeStringsEqual(barTimeString(row), expectedTime) {
			return idx, row, row != nil
		}
	}
	if expectedTime != "" {
		for j, raw := range chartBars {
			r, _ := raw.(map[string]any)
			if barTimeStringsEqual(barTimeString(r), expectedTime) {
				return j, r, true
			}
		}
	}
	if idx < len(chartBars) {
		row = barRowAt(chartBars, idx)
		return idx, row, row != nil
	}
	return 0, nil, false
}

func labDetailInputChartBars(in LabDetailInput) []any {
	if chart := LabDetailChartBars(in.ProbeRaw); len(chart) > 0 {
		return chart
	}
	return in.Bars
}
