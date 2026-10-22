package strategydev

import (
	"context"
	"time"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func init() {
	kp.SetHost(kp.Host{
		ProbeArgsExtended: func(code, freq string, months int, asOf *time.Time) map[string]any {
			return PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{
				Code: code, Freq: freq, Months: months, AsOf: asOf, ExtendedFetch: true,
			})
		},
		MonthsBackForPerSignalFetch: monthsBackForPerSignalPriceFetch,
	})
}

// KlinePatternDisplayInput is archived lab display context (re-exported).
type KlinePatternDisplayInput = kp.KlinePatternDisplayInput

// KlinePatternMTFBundle is archived MTF probe bundle (re-exported).
type KlinePatternMTFBundle = kp.KlinePatternMTFBundle

// KlinePatternFreqSeries is one TF series in the MTF bundle (re-exported).
type KlinePatternFreqSeries = kp.KlinePatternFreqSeries

// NewKlinePatternDisplayInput builds probe context when display is enabled.
func NewKlinePatternDisplayInput(signalFreq string, bars []any, probeRaw map[string]any, mtf *KlinePatternMTFBundle) KlinePatternDisplayInput {
	if !KlinePatternDisplayEnabled() {
		return kp.KlinePatternDisplayInput{}
	}
	return kp.NewKlinePatternDisplayInput(signalFreq, bars, probeRaw, mtf)
}

// NewKlinePatternDisplayInputForDiagnose builds probe context when signal-diagnose lab_mode is on.
func NewKlinePatternDisplayInputForDiagnose(labMode bool, signalFreq string, bars []any, probeRaw map[string]any, mtf *KlinePatternMTFBundle) KlinePatternDisplayInput {
	if !DiagnoseLabMode(labMode) {
		return kp.KlinePatternDisplayInput{}
	}
	return kp.NewKlinePatternDisplayInput(signalFreq, bars, probeRaw, mtf)
}

// BuildKlinePatternMTFForDiagnose fetches MTF probes for diagnose when display is enabled.
func BuildKlinePatternMTFForDiagnose(
	ctx context.Context,
	run ToolRunner,
	code, signalFreq string,
	months int,
	bars []any,
) *KlinePatternMTFBundle {
	if !KlinePatternDisplayEnabled() || run == nil || len(bars) == 0 {
		return nil
	}
	return kp.BuildKlinePatternMTFBundle(ctx, kp.ToolRunner(run), code, signalFreq, months, bars)
}

// BuildKlinePatternMTFForDiagnoseRequest fetches MTF probes when signal-diagnose lab_mode is on.
func BuildKlinePatternMTFForDiagnoseRequest(
	labMode bool,
	ctx context.Context,
	run ToolRunner,
	code, signalFreq string,
	months int,
	bars []any,
) *KlinePatternMTFBundle {
	if !DiagnoseLabMode(labMode) || run == nil || len(bars) == 0 {
		return nil
	}
	return kp.BuildKlinePatternMTFBundleParallel(ctx, kp.ToolRunner(run), code, signalFreq, months, bars)
}

// BuildKlinePatternMTFBundle probes higher timeframes (lab archive; opt-in only).
func BuildKlinePatternMTFBundle(
	ctx context.Context,
	run ToolRunner,
	code, signalFreq string,
	months int,
	chartBars []any,
) *KlinePatternMTFBundle {
	return kp.BuildKlinePatternMTFBundle(ctx, kp.ToolRunner(run), code, signalFreq, months, chartBars)
}

// EnrichSignalEpisodeEvalKlineDisplay attaches archived pattern display fields to episodes.
func EnrichSignalEpisodeEvalKlineDisplay(eval *slots.SignalEpisodeEval, in KlinePatternDisplayInput) {
	if !KlinePatternDisplayEnabled() {
		return
	}
	kp.EnrichSignalEpisodeEvalKlineDisplay(eval, in)
}

// EnrichSignalEpisodeEvalKlineDisplayLight attaches signal-TF summaries only (lab_mode first screen).
func EnrichSignalEpisodeEvalKlineDisplayLight(eval *slots.SignalEpisodeEval, in KlinePatternDisplayInput) {
	if eval == nil || len(in.PatternHits) == 0 && in.MTF == nil {
		return
	}
	kp.EnrichSignalEpisodeEvalKlineDisplayLight(eval, in)
}

// EnrichEpisodeTriggerFilterStatsKlineDisplay enriches trigger lists when display is enabled.
func EnrichEpisodeTriggerFilterStatsKlineDisplay(stats *EpisodeTriggerFilterStats, in KlinePatternDisplayInput) {
	if stats == nil || !KlinePatternDisplayEnabled() {
		return
	}
	apply := func(list *[]EpisodeTriggerRecord) {
		for i := range *list {
			enrichTriggerKlineDisplay(&(*list)[i], in)
		}
	}
	apply(&stats.BuyTriggers)
	apply(&stats.SellTriggers)
	apply(&stats.StrippedBuyTriggers)
	apply(&stats.StrippedSellTriggers)
	apply(&stats.KeptBuyTriggers)
	apply(&stats.KeptSellTriggers)
}

// AttachKlineDisplayToProbeTriggers enriches triggers with full display context.
func AttachKlineDisplayToProbeTriggers(buy, sell []EpisodeTriggerRecord, in KlinePatternDisplayInput) {
	if !KlinePatternDisplayEnabled() {
		return
	}
	for i := range buy {
		enrichTriggerKlineDisplay(&buy[i], in)
	}
	for i := range sell {
		enrichTriggerKlineDisplay(&sell[i], in)
	}
}

func enrichTriggerKlineDisplayForDiagnose(labMode bool, rec *EpisodeTriggerRecord, in KlinePatternDisplayInput) {
	if rec == nil || !DiagnoseLabMode(labMode) {
		return
	}
	enrichEpisodeTriggerRecordPatterns(rec, in.PatternHits)
	rec.KlinePatternThreeBar = kp.BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, rec.Index)
	rec.KlinePatternMTF = kp.BuildKlinePatternMTFContext(rec.Index, in)
	if rec.KlinePatternMTF != nil {
		rec.HigherTFContext = rec.KlinePatternMTF
		kp.ApplyMTFWeightedKlineConclusion(rec.KlinePatternMTF, &rec.KlinePatternBias, &rec.KlinePatternSummary)
	}
	if rec.KlinePatternBias == nil {
		if rec.KlinePatternThreeBar != nil && rec.KlinePatternThreeBar.Bias != nil {
			rec.KlinePatternBias = rec.KlinePatternThreeBar.Bias
		} else if len(rec.KlinePatterns) > 0 {
			b := slots.ScoreKlinePatternHits(rec.KlinePatterns)
			rec.KlinePatternBias = &b
		}
	}
}

func enrichTriggerKlineDisplay(rec *EpisodeTriggerRecord, in KlinePatternDisplayInput) {
	if rec == nil {
		return
	}
	enrichEpisodeTriggerRecordPatterns(rec, in.PatternHits)
	rec.KlinePatternThreeBar = kp.BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, rec.Index)
	rec.KlinePatternMTF = kp.BuildKlinePatternMTFContext(rec.Index, in)
	if rec.KlinePatternMTF != nil {
		rec.HigherTFContext = rec.KlinePatternMTF
		kp.ApplyMTFWeightedKlineConclusion(rec.KlinePatternMTF, &rec.KlinePatternBias, &rec.KlinePatternSummary)
	}
	if rec.KlinePatternBias == nil {
		if rec.KlinePatternThreeBar != nil && rec.KlinePatternThreeBar.Bias != nil {
			rec.KlinePatternBias = rec.KlinePatternThreeBar.Bias
		} else if len(rec.KlinePatterns) > 0 {
			b := slots.ScoreKlinePatternHits(rec.KlinePatterns)
			rec.KlinePatternBias = &b
		}
	}
}

// EnrichEpisodeTriggerRecordPatterns fills kline_patterns on one trigger record.
func EnrichEpisodeTriggerRecordPatterns(rec *EpisodeTriggerRecord, series []any) {
	if rec == nil || !KlinePatternDisplayEnabled() {
		return
	}
	enrichEpisodeTriggerRecordPatterns(rec, series)
}

func enrichEpisodeTriggerRecordPatterns(rec *EpisodeTriggerRecord, series []any) {
	if rec == nil {
		return
	}
	hits := kp.FilteredKlinePatternsAtIndex(series, rec.Index)
	if len(hits) == 0 {
		rec.KlinePatterns = nil
		rec.KlinePatternSummary = ""
		return
	}
	rec.KlinePatterns = hits
	rec.KlinePatternSummary = slots.KlinePatternSummary(hits)
}
