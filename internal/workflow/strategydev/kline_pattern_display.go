package strategydev

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// KlinePatternDisplayInput is probe context for enriching triggers/episodes.
type KlinePatternDisplayInput struct {
	SignalFreq  string
	Bars        []any
	PatternHits []any
	MTF         *KlinePatternMTFBundle
}

// NewKlinePatternDisplayInput builds display input from probe raw map and optional MTF bundle.
func NewKlinePatternDisplayInput(signalFreq string, bars []any, probeRaw map[string]any, mtf *KlinePatternMTFBundle) KlinePatternDisplayInput {
	var hits []any
	if probeRaw != nil {
		hits = PatternHitsSeriesFromProbe(probeRaw)
	}
	return KlinePatternDisplayInput{
		SignalFreq:  signalFreq,
		Bars:        bars,
		PatternHits: hits,
		MTF:         mtf,
	}
}

// BuildKlinePatternMTFForDiagnose fetches MTF series when runner is available.
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
	return BuildKlinePatternMTFBundle(ctx, run, code, signalFreq, months, bars)
}

func enrichTriggerKlineDisplay(rec *EpisodeTriggerRecord, in KlinePatternDisplayInput) {
	if rec == nil || !KlinePatternDisplayEnabled() {
		return
	}
	EnrichEpisodeTriggerRecordPatterns(rec, in.PatternHits)
	rec.KlinePatternThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, rec.Index)
	if rec.KlinePatternThreeBar != nil && rec.KlinePatternThreeBar.Bias != nil {
		rec.KlinePatternBias = rec.KlinePatternThreeBar.Bias
	} else if len(rec.KlinePatterns) > 0 {
		b := slots.ScoreKlinePatternHits(rec.KlinePatterns)
		rec.KlinePatternBias = &b
	}
	rec.KlinePatternMTF = buildKlinePatternMTFContext(rec.Index, in)
	if rec.KlinePatternMTF != nil {
		rec.HigherTFContext = rec.KlinePatternMTF
	}
}

func enrichEpisodeKlineDisplay(ep *slots.SignalEpisodeDetail, in KlinePatternDisplayInput) {
	if ep == nil || !KlinePatternDisplayEnabled() {
		return
	}
	attachPatternsToEpisode(ep, in.PatternHits)
	ep.KlinePatternThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, ep.StartIdx)
	if ep.KlinePatternThreeBar != nil && ep.KlinePatternThreeBar.Bias != nil {
		ep.KlinePatternBias = ep.KlinePatternThreeBar.Bias
	} else if len(ep.KlinePatterns) > 0 {
		b := slots.ScoreKlinePatternHits(ep.KlinePatterns)
		ep.KlinePatternBias = &b
	}
	ep.KlinePatternMTF = buildKlinePatternMTFContext(ep.StartIdx, in)
	if ep.KlinePatternMTF != nil {
		ep.HigherTFContext = ep.KlinePatternMTF
	}
}

func buildKlinePatternMTFContext(anchorIdx int, in KlinePatternDisplayInput) *slots.KlinePatternMTFContext {
	if in.MTF == nil || len(in.MTF.ByFreq) == 0 {
		return nil
	}
	trigger := triggerTimeFromBars(in.Bars, anchorIdx)
	if trigger.IsZero() {
		return nil
	}
	ctxOut := &slots.KlinePatternMTFContext{
		SignalFrequency: NormalizeFrequency(in.SignalFreq),
		Note:            "大周期仅作背景对照，不参与拦截或硬结论",
	}
	sigFreq := NormalizeFrequency(in.SignalFreq)
	for _, spec := range multiTimeframeSpecsForGate(in.SignalFreq) {
		entry := slots.KlinePatternMTFEntry{
			Frequency:   spec.Freq,
			Label:       spec.Label,
			DisplayOnly: true,
		}
		if NormalizeFrequency(spec.Freq) == sigFreq {
			entry.FetchOK = len(in.Bars) > 0
			entry.AlignedIndex = anchorIdx
			if anchorIdx >= 0 && anchorIdx < len(in.Bars) {
				if row, ok := in.Bars[anchorIdx].(map[string]any); ok {
					entry.AlignedTime = formatBarTimeAsiaShanghai(row)
				}
				entry.ThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, anchorIdx)
				if entry.ThreeBar != nil && entry.ThreeBar.Bias != nil {
					entry.Bias = entry.ThreeBar.Bias
				}
			}
			entry.AlignNote = mtfAlignNote(spec.Freq, anchorIdx, entry.FetchOK)
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		ser, ok := in.MTF.ByFreq[spec.Freq]
		if !ok {
			entry.FetchOK = false
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, false)
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		entry.FetchOK = ser.FetchOK
		if !ser.FetchOK {
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, false)
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		aligned := alignBarIndexAtOrBefore(ser.Bars, trigger)
		entry.AlignedIndex = aligned
		if aligned >= 0 {
			if row, ok := ser.Bars[aligned].(map[string]any); ok {
				entry.AlignedTime = formatBarTimeAsiaShanghai(row)
			}
			entry.ThreeBar = BuildKlinePatternThreeBarWindow(ser.Bars, ser.PatternHits, aligned)
			if entry.ThreeBar != nil && entry.ThreeBar.Bias != nil {
				entry.Bias = entry.ThreeBar.Bias
			}
		}
		entry.AlignNote = mtfAlignNote(spec.Freq, aligned, true)
		ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
	}
	if win := BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, anchorIdx); win != nil && win.Bias != nil {
		ctxOut.PrimaryBias = win.Bias
	}
	return ctxOut
}

// EnrichSignalEpisodeEvalKlineDisplay attaches 3-bar window, MTF, and bias to episodes.
func EnrichSignalEpisodeEvalKlineDisplay(eval *slots.SignalEpisodeEval, in KlinePatternDisplayInput) {
	if eval == nil || !KlinePatternDisplayEnabled() {
		return
	}
	for i := range eval.BuyDetails {
		enrichEpisodeKlineDisplay(&eval.BuyDetails[i], in)
	}
	for i := range eval.SellDetails {
		enrichEpisodeKlineDisplay(&eval.SellDetails[i], in)
	}
}

// EnrichEpisodeTriggerFilterStatsKlineDisplay enriches all trigger lists.
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
	for i := range buy {
		enrichTriggerKlineDisplay(&buy[i], in)
	}
	for i := range sell {
		enrichTriggerKlineDisplay(&sell[i], in)
	}
}
