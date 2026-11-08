package klinepattern

import (
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
	hits := patternHitsForDisplayInput(signalFreq, bars, probeRaw, mtf)
	return KlinePatternDisplayInput{
		SignalFreq:  signalFreq,
		Bars:        bars,
		PatternHits: hits,
		MTF:         mtf,
	}
}

// patternHitsForDisplayInput prefers probe_raw.pattern_hits; falls back to MTF signal-TF probe when
// chart_probe persistence omitted pattern_hits (macro lab-detail / SignalDiagnoseResult rebuild).
func patternHitsForDisplayInput(signalFreq string, chartBars []any, probeRaw map[string]any, mtf *KlinePatternMTFBundle) []any {
	sig := NormalizeFrequency(signalFreq)
	// Macro daily_macro: chart bars are signal TF (60m) but MTF bundle probes daily patterns — prefer daily series.
	if mtf != nil && NormalizeFrequency(mtf.SignalFreq) == sig && len(chartBars) > 0 {
		if ser, ok := mtf.ByFreq[klinePatternMTFTimeframeKey(sig)]; ok && len(ser.PatternHits) > 0 {
			if len(ser.PatternHits) == len(chartBars) {
				return ser.PatternHits
			}
			return AlignPatternHitsSeriesToBars(chartBars, ser.Bars, ser.PatternHits)
		}
	}
	if probeRaw != nil {
		if hits := PatternHitsSeriesFromProbe(probeRaw); len(hits) > 0 {
			if len(chartBars) == 0 || len(hits) == len(chartBars) {
				return hits
			}
		}
	}
	if mtf == nil || len(chartBars) == 0 {
		return nil
	}
	ser, ok := mtf.ByFreq[klinePatternMTFTimeframeKey(sig)]
	if !ok || len(ser.PatternHits) == 0 {
		return nil
	}
	if len(ser.PatternHits) == len(chartBars) {
		return ser.PatternHits
	}
	return AlignPatternHitsSeriesToBars(chartBars, ser.Bars, ser.PatternHits)
}

// EnrichEpisodeDetailKlineDisplay attaches 3-bar window, MTF, and bias to one episode row.
func EnrichEpisodeDetailKlineDisplay(ep *slots.SignalEpisodeDetail, in KlinePatternDisplayInput) {
	if ep == nil {
		return
	}
	AttachPatternsToEpisode(ep, in.PatternHits)
	ep.KlinePatternThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, ep.StartIdx)
	ep.KlinePatternMTF = BuildKlinePatternMTFContext(ep.StartIdx, in)
	if ep.KlinePatternMTF != nil {
		ep.HigherTFContext = ep.KlinePatternMTF
		ApplyMTFWeightedKlineConclusion(ep.KlinePatternMTF, &ep.KlinePatternBias, &ep.KlinePatternSummary)
	}
	if ep.KlinePatternBias == nil {
		if ep.KlinePatternThreeBar != nil && ep.KlinePatternThreeBar.Bias != nil {
			ep.KlinePatternBias = ep.KlinePatternThreeBar.Bias
		} else if len(ep.KlinePatterns) > 0 {
			b := slots.ScoreKlinePatternHits(ep.KlinePatterns)
			ep.KlinePatternBias = &b
		}
	}
}

// ApplyMTFWeightedKlineConclusion sets primary_bias and optional trigger-level bias/summary.
func ApplyMTFWeightedKlineConclusion(mtf *slots.KlinePatternMTFContext, bias **slots.KlinePatternBias, summary *string) {
	if mtf == nil || len(mtf.Timeframes) == 0 {
		return
	}
	b, sum, ok := ComputeKlinePatternMTFWeightedConclusion(mtf.Timeframes)
	if !ok {
		return
	}
	mtf.WeightSchema = KlinePatternMTFWeightSchema
	mtf.PrimaryBias = &b
	if bias != nil {
		*bias = &b
	}
	if summary != nil && sum != "" {
		*summary = sum
	}
}

// SignalOnlyKlinePatternMTFBundle is an empty MTF probe bundle for light enrich (signal TF row only).
func SignalOnlyKlinePatternMTFBundle(signalFreq string) *KlinePatternMTFBundle {
	return &KlinePatternMTFBundle{
		SignalFreq: NormalizeFrequency(signalFreq),
		ByFreq:     map[string]KlinePatternFreqSeries{},
	}
}

// BuildKlinePatternMTFContext assembles MTF JSON for one anchor index.
func BuildKlinePatternMTFContext(anchorIdx int, in KlinePatternDisplayInput) *slots.KlinePatternMTFContext {
	if in.MTF == nil {
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
	for _, spec := range multiTimeframeSpecsForKlinePatternDisplay(in.SignalFreq) {
		entry := slots.KlinePatternMTFEntry{
			Frequency:   spec.Freq,
			Label:       spec.Label,
			DisplayOnly: true,
		}
		if klinePatternMTFTimeframeKey(spec.Freq) == klinePatternMTFTimeframeKey(sigFreq) {
			// Daily macro may set FetchOK false when pattern_hits sparse; still align on probed TF bars.
			if ser, ok := mtfBundleSeries(in.MTF, spec.Freq); ok && len(ser.Bars) > 0 {
				aligned := alignBarIndexAtOrBefore(ser.Bars, trigger)
				entry.FetchOK = true
				entry.AlignedIndex = aligned
				if aligned >= 0 && aligned < len(ser.Bars) {
					if row, ok := ser.Bars[aligned].(map[string]any); ok {
						entry.AlignedTime = formatBarTimeAsiaShanghai(row)
					}
					entry.ThreeBar = BuildKlinePatternThreeBarWindow(ser.Bars, ser.PatternHits, aligned)
				}
				entry.AlignNote = mtfAlignNote(spec.Freq, aligned, mtfAlignOK)
				b := biasFromThreeBarWindow(entry.ThreeBar)
				entry.Bias = &b
				ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
				continue
			}
			entry.FetchOK = len(in.Bars) > 0
			entry.AlignedIndex = anchorIdx
			if anchorIdx >= 0 && anchorIdx < len(in.Bars) {
				if row, ok := in.Bars[anchorIdx].(map[string]any); ok {
					entry.AlignedTime = formatBarTimeAsiaShanghai(row)
				}
				entry.ThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, anchorIdx)
			}
			st := mtfAlignOK
			if !entry.FetchOK {
				st = mtfAlignFetchFailed
			}
			entry.AlignNote = mtfAlignNote(spec.Freq, anchorIdx, st)
			b := biasFromThreeBarWindow(entry.ThreeBar)
			entry.Bias = &b
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		ser, ok := mtfBundleSeries(in.MTF, spec.Freq)
		if !ok {
			entry.FetchOK = false
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, mtfAlignMissingSeries)
			nb := neutralKlinePatternBias()
			entry.Bias = &nb
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		entry.FetchOK = ser.FetchOK
		if !ser.FetchOK {
			st := mtfAlignFetchFailed
			if ser.Unsupported {
				st = mtfAlignUnsupportedFreq
			}
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, st)
			nb := neutralKlinePatternBias()
			entry.Bias = &nb
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
		}
		entry.AlignNote = mtfAlignNote(spec.Freq, aligned, mtfAlignOK)
		b := biasFromThreeBarWindow(entry.ThreeBar)
		entry.Bias = &b
		ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
	}
	return ctxOut
}

// EnrichSignalEpisodeEvalKlineDisplay attaches 3-bar window, MTF, and bias to episodes.
func EnrichSignalEpisodeEvalKlineDisplay(eval *slots.SignalEpisodeEval, in KlinePatternDisplayInput) {
	if eval == nil {
		return
	}
	for i := range eval.BuyDetails {
		EnrichEpisodeDetailKlineDisplay(&eval.BuyDetails[i], in)
	}
	for i := range eval.SellDetails {
		EnrichEpisodeDetailKlineDisplay(&eval.SellDetails[i], in)
	}
}

// EnrichEpisodeDetailKlineDisplayLight attaches signal-TF patterns and three-bar window only.
// Multi-TF kline context is lab-detail lazy output (see BuildLabEpisodeDetail), not workflow cards.
func EnrichEpisodeDetailKlineDisplayLight(ep *slots.SignalEpisodeDetail, in KlinePatternDisplayInput) {
	if ep == nil {
		return
	}
	AttachPatternsToEpisode(ep, in.PatternHits)
	ep.KlinePatternThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, ep.StartIdx)
	if ep.KlinePatternBias == nil {
		if ep.KlinePatternThreeBar != nil && ep.KlinePatternThreeBar.Bias != nil {
			ep.KlinePatternBias = ep.KlinePatternThreeBar.Bias
		} else if len(ep.KlinePatterns) > 0 {
			b := slots.ScoreKlinePatternHits(ep.KlinePatterns)
			ep.KlinePatternBias = &b
		}
	}
}

// EnrichSignalEpisodeEvalKlineDisplayLight attaches signal-TF display fields without MTF probes.
func EnrichSignalEpisodeEvalKlineDisplayLight(eval *slots.SignalEpisodeEval, in KlinePatternDisplayInput) {
	if eval == nil {
		return
	}
	for i := range eval.BuyDetails {
		EnrichEpisodeDetailKlineDisplayLight(&eval.BuyDetails[i], in)
	}
	for i := range eval.SellDetails {
		EnrichEpisodeDetailKlineDisplayLight(&eval.SellDetails[i], in)
	}
}
