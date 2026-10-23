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
			entry.FetchOK = len(in.Bars) > 0
			entry.AlignedIndex = anchorIdx
			if anchorIdx >= 0 && anchorIdx < len(in.Bars) {
				if row, ok := in.Bars[anchorIdx].(map[string]any); ok {
					entry.AlignedTime = formatBarTimeAsiaShanghai(row)
				}
				entry.ThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, anchorIdx)
			}
			entry.AlignNote = mtfAlignNote(spec.Freq, anchorIdx, entry.FetchOK)
			b := biasFromThreeBarWindow(entry.ThreeBar)
			entry.Bias = &b
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		ser, ok := in.MTF.ByFreq[spec.Freq]
		if !ok {
			entry.FetchOK = false
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, false)
			nb := neutralKlinePatternBias()
			entry.Bias = &nb
			ctxOut.Timeframes = append(ctxOut.Timeframes, entry)
			continue
		}
		entry.FetchOK = ser.FetchOK
		if !ser.FetchOK {
			entry.AlignNote = mtfAlignNote(spec.Freq, -1, false)
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
		entry.AlignNote = mtfAlignNote(spec.Freq, aligned, true)
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

// EnrichEpisodeDetailKlineDisplayLight attaches signal-TF patterns, three-bar window, and MTF shell (no higher-TF probes).
func EnrichEpisodeDetailKlineDisplayLight(ep *slots.SignalEpisodeDetail, in KlinePatternDisplayInput) {
	if ep == nil {
		return
	}
	AttachPatternsToEpisode(ep, in.PatternHits)
	ep.KlinePatternThreeBar = BuildKlinePatternThreeBarWindow(in.Bars, in.PatternHits, ep.StartIdx)
	shellIn := in
	if shellIn.MTF == nil && len(in.Bars) > 0 {
		shellIn.MTF = SignalOnlyKlinePatternMTFBundle(in.SignalFreq)
	}
	if shellIn.MTF != nil {
		ep.KlinePatternMTF = BuildKlinePatternMTFContext(ep.StartIdx, shellIn)
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
