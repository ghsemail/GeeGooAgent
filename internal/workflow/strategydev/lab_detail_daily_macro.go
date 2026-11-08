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

// prepareDailyMacroKlineProbe ensures the MTF bundle has daily bars (probe or lab-api) before kline display.
func prepareDailyMacroKlineProbe(
	ctx context.Context,
	in LabDetailInput,
	chartBars []any,
	chartIdx int,
	mtf *KlinePatternMTFBundle,
) {
	if mtf == nil {
		return
	}
	dailyBars := dailyBarsForLabAnchorFromMTF(mtf)
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

func dailyMacroKlineDisplayInput(
	chartBars []any,
	chartIdx int,
	probeRaw map[string]any,
	mtf *KlinePatternMTFBundle,
) (KlinePatternDisplayInput, int) {
	dailyBars := dailyBarsForLabAnchorFromMTF(mtf)
	if len(dailyBars) == 0 {
		return NewKlinePatternDisplayInputForDiagnose(true, "daily", chartBars, probeRaw, mtf), chartIdx
	}
	idx := kp.AlignAnchorBarIndex(chartBars, chartIdx, dailyBars)
	return NewKlinePatternDisplayInputForDiagnose(true, "daily", dailyBars, probeRaw, mtf), idx
}
