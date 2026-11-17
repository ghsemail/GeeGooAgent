package strategydev

import (
	"context"
	"strings"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
)

func labDetailDailyMacro(in LabDetailInput) bool {
	return strings.TrimSpace(in.KlineJEVProfile) == LabKlineJEVProfileDailyMacro
}

// labDetailJEVSignalFreq is the TF label passed into JEV state for macro (daily only).
// dailyMacroAnalysisGated is false when daily_macro must fail-closed without a completed session cutoff.
func dailyMacroAnalysisGated(dailyMacro, cutoffOK bool) bool {
	return !(dailyMacro && !cutoffOK)
}

func labDetailJEVSignalFreq(in LabDetailInput) string {
	if labDetailDailyMacro(in) {
		return "daily"
	}
	return strings.TrimSpace(in.SignalFreq)
}

// LabDetailCacheRef keys persisted lab-detail (episode ref + optional kline JEV profile).
func LabDetailCacheRef(detailRef, klineJEVProfile string) string {
	detailRef = strings.TrimSpace(detailRef)
	profile := strings.TrimSpace(klineJEVProfile)
	if profile == "" {
		return detailRef
	}
	return detailRef + "|" + profile
}

func LabDetailProfileMatches(cached LabEpisodeDetail, wantProfile string) bool {
	return strings.TrimSpace(cached.KlineJEVProfile) == strings.TrimSpace(wantProfile)
}

// prepareDailyMacroKlineProbe ensures the MTF bundle has daily bars (+ pattern_hits when possible).
// dailyBarsHint may be pre-fetched lab-api series (PA budget); otherwise fetches when LabFetcher is set.
func prepareDailyMacroKlineProbe(
	ctx context.Context,
	in LabDetailInput,
	chartBars []any,
	mtf *KlinePatternMTFBundle,
	dailyBarsHint []any,
) {
	if mtf == nil {
		return
	}
	dailyBars := dailyBarsHint
	if len(dailyBars) == 0 {
		dailyBars = dailyBarsForLabAnchorFromMTF(mtf)
	}
	if len(dailyBars) == 0 && in.LabFetcher != nil {
		dailyBars = dailyBarsForLabAnchor(ctx, in)
	}
	if len(dailyBars) == 0 {
		return
	}
	if in.Run != nil {
		kp.HydrateDailyMacroMTFFromBars(ctx, kp.ToolRunner(in.Run), in.Code, in.Months, dailyBars, mtf)
		return
	}
	key := kp.NormalizeFrequency("daily")
	if ser, ok := mtf.ByFreq[key]; !ok || len(ser.Bars) == 0 {
		if mtf.ByFreq == nil {
			mtf.ByFreq = map[string]kp.KlinePatternFreqSeries{}
		}
		mtf.SignalFreq = "daily"
		mtf.ByFreq[key] = kp.KlinePatternFreqSeries{
			Label: "日线", Freq: "daily", Bars: dailyBars, FetchOK: true,
		}
	}
}

// dailyMacroKlineDisplayInputAtCutoff builds daily kline/three_bar at the macro monitor bar day,
// not at historical buy/signal chartIdx.
func dailyMacroKlineDisplayInputAtCutoff(
	code string,
	cut MacroMonitorCutoff,
	probeRaw map[string]any,
	mtf *KlinePatternMTFBundle,
) (KlinePatternDisplayInput, int) {
	dailyBars := dailyBarsForLabAnchorFromMTF(mtf)
	if len(dailyBars) == 0 {
		return NewKlinePatternDisplayInputForDiagnose(true, "daily", nil, probeRaw, mtf), 0
	}
	spec, ok := macroSessionSpecForCode(code)
	if !ok {
		return NewKlinePatternDisplayInputForDiagnose(true, "daily", dailyBars, probeRaw, mtf), len(dailyBars) - 1
	}
	idx := indexDailyBarByCalendarDay(dailyBars, cut.CalendarDay, spec)
	if idx < 0 {
		idx = len(dailyBars) - 1
	}
	return NewKlinePatternDisplayInputForDiagnose(true, "daily", dailyBars, probeRaw, mtf), idx
}
