package strategydev

import (
	"testing"
	"time"
)

func TestDailyMacroKlineDisplayInputAtCutoff_IgnoresHistoricalChartIdx(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 11, 16, 16, 30, 0, 0, loc)
	dailyBars := []any{
		map[string]any{"time": "2026-11-03 00:00:00", "close": 100.0},
		map[string]any{"time": "2026-11-16 00:00:00", "close": 200.0},
	}
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("000300.SH", now, dailyBars)
	if !ok {
		t.Fatal("cutoff")
	}
	mtf := &KlinePatternMTFBundle{
		SignalFreq: "daily",
		ByFreq: map[string]KlinePatternFreqSeries{
			"daily": {Freq: "daily", Bars: dailyBars, FetchOK: true},
		},
	}
	chartBars := []any{
		map[string]any{"time": "2026-11-03 10:00:00", "close": 99.0},
	}
	historicalBuyIdx := 0
	_, idxAtCut := dailyMacroKlineDisplayInputAtCutoff("000300.SH", cut, nil, mtf)
	if idxAtCut != 1 {
		t.Fatalf("kline idx=%d want 1 (monitor day %s)", idxAtCut, cut.CalendarDay)
	}
	_, legacyIdx := dailyMacroKlineDisplayInputLegacyAlign(chartBars, historicalBuyIdx, nil, mtf)
	if legacyIdx == idxAtCut {
		t.Fatalf("legacy align must not equal cutoff idx when buy predates monitor day")
	}
}

// dailyMacroKlineDisplayInputLegacyAlign documents rejected historical-buy alignment (C1).
func dailyMacroKlineDisplayInputLegacyAlign(
	chartBars []any,
	chartIdx int,
	probeRaw map[string]any,
	mtf *KlinePatternMTFBundle,
) (KlinePatternDisplayInput, int) {
	dailyBars := dailyBarsForLabAnchorFromMTF(mtf)
	if len(dailyBars) == 0 {
		return NewKlinePatternDisplayInputForDiagnose(true, "daily", chartBars, probeRaw, mtf), chartIdx
	}
	idx := alignDailyBarIndexFromChart(chartBars, chartIdx, dailyBars)
	return NewKlinePatternDisplayInputForDiagnose(true, "daily", dailyBars, probeRaw, mtf), idx
}

func alignDailyBarIndexFromChart(chartBars []any, chartIdx int, dailyBars []any) int {
	if chartIdx < 0 || chartIdx >= len(chartBars) {
		return len(dailyBars) - 1
	}
	trigger := barTimeUnix(chartBars[chartIdx].(map[string]any))
	for i, raw := range dailyBars {
		row := raw.(map[string]any)
		if !barTimeUnix(row).After(trigger) {
			continue
		}
		return i - 1
	}
	return len(dailyBars) - 1
}

func TestDailyMacroAnalysisGated_FailClosedWithoutCutoff(t *testing.T) {
	if dailyMacroAnalysisGated(true, false) {
		t.Fatal("daily_macro must not run PA/TA/JEV without cutoff")
	}
	if !dailyMacroAnalysisGated(true, true) {
		t.Fatal("expected analysis when cutoff ok")
	}
	if !dailyMacroAnalysisGated(false, false) {
		t.Fatal("non-macro paths keep legacy behavior")
	}
}
