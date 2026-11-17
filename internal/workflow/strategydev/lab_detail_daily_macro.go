package strategydev

import (
	"context"
	"strings"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
)

const (
	// LabDetailDailyMacroCacheContract bumps when daily_macro lab-detail JSON contract changes (cache invalidation).
	LabDetailDailyMacroCacheContract = 2
)

func labDetailDailyMacro(in LabDetailInput) bool {
	return strings.TrimSpace(in.KlineJEVProfile) == LabKlineJEVProfileDailyMacro
}

// LabDetailCacheAcceptable returns false for stale daily_macro payloads missing session-close contract fields.
func LabDetailCacheAcceptable(cached LabEpisodeDetail, wantProfile string) bool {
	wantProfile = strings.TrimSpace(wantProfile)
	if wantProfile != LabKlineJEVProfileDailyMacro {
		return true
	}
	if !LabDetailProfileMatches(cached, wantProfile) {
		return false
	}
	if cached.LabDetailCacheContract < LabDetailDailyMacroCacheContract {
		return false
	}
	if strings.TrimSpace(cached.MonitorSessionClose) == "" {
		return false
	}
	return true
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
	mtf *KlinePatternMTFBundle,
	dailyBarsHint []any,
	cut MacroMonitorCutoff,
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
		kp.HydrateDailyMacroMTFFromBarsForCalendarDay(ctx, kp.ToolRunner(in.Run), in.Code, in.Months, dailyBars, mtf, cut.CalendarDay)
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
) (KlinePatternDisplayInput, int, bool) {
	dailyBars := dailyBarsForLabAnchorFromMTF(mtf)
	if len(dailyBars) == 0 {
		return KlinePatternDisplayInput{}, -1, false
	}
	spec, ok := macroSessionSpecForCode(code)
	if !ok {
		return KlinePatternDisplayInput{}, -1, false
	}
	idx := indexDailyBarByCalendarDayExact(dailyBars, cut.CalendarDay, spec)
	if idx < 0 {
		return KlinePatternDisplayInput{}, -1, false
	}
	patIn := NewKlinePatternDisplayInputForDiagnose(true, "daily", dailyBars, probeRaw, mtf)
	return patIn, idx, true
}
