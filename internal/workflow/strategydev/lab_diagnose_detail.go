package strategydev

import (
	"context"
	"strings"
	"time"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
)

const (
	// DefaultLabDetailPhaseTimeout bounds kline MTF probe for one lab-detail call.
	DefaultLabDetailPhaseTimeout = 8 * time.Second
	// DefaultLabDetailPATimeout is a separate budget for lab-api multi-TF price analysis (after MTF probe).
	DefaultLabDetailPATimeout = 45 * time.Second
	// DefaultLabDetailJEVTimeout is the dedicated System One budget after PA completes (matches JevClient HTTP timeout).
	DefaultLabDetailJEVTimeout = 45 * time.Second
)

// LabDetailInput configures lazy lab drill-down for one trigger/episode anchor.
type LabDetailInput struct {
	Run           ToolRunner
	Code          string
	SignalFreq    string
	Months        int
	KeyLevels     KeyLevelBands
	ProbeRaw      map[string]any
	Bars          []any
	LookbackBars  []any
	LabFetcher    priceanalysis.BarsFetcher
	// DetailTimeout caps kline MTF probe only (not PA / JEV).
	DetailTimeout time.Duration
	// PATimeout caps lab-api price-analysis fetch (separate from MTF probe).
	PATimeout time.Duration
	// JEVTimeout is a separate budget for System One price-analysis JEV after PA finishes.
	JEVTimeout time.Duration
	// CachedMTF when set, reuses MTF bundle across lab-detail calls for one diagnose run.
	CachedMTF **KlinePatternMTFBundle
	SessionID string
	RunID     string
	// Jev optional System One client for lazy price-analysis JEV (lab-detail only).
	Jev *JevClient
}

// BuildLabEpisodeDetail loads MTF kline context and anchored price analysis for one ref.
func BuildLabEpisodeDetail(ctx context.Context, in LabDetailInput, ref string) (LabEpisodeDetail, error) {
	side, idx, ok := ParseLabDetailRef(ref)
	if !ok {
		return LabEpisodeDetail{}, errBadLabRef("invalid detail ref")
	}
	if side == "flow" {
		return buildLabFlowDetail(ctx, in, ref)
	}
	out := LabEpisodeDetail{Ref: ref, Side: side, BarIndex: idx}
	if len(in.Bars) == 0 || idx >= len(in.Bars) {
		return out, errBadLabRef("bar index out of range: detail_ref suffix is chart start_idx (see lab_episode_summaries.start_idx), not episode list index")
	}
	ent := triggerRecordEntry(in.Bars, idx)
	out.Time = ent.Time
	out.Close = ent.Close

	cacheCtx := ContextWithPriceProbeCache(ctx)
	phaseCtx, phaseCancel := labDetailPhaseContext(cacheCtx, in.DetailTimeout)
	defer phaseCancel()

	var mtf *KlinePatternMTFBundle
	if in.CachedMTF != nil && *in.CachedMTF != nil {
		mtf = *in.CachedMTF
	} else if b := CachedLabKlineMTF(in.SessionID, in.RunID); b != nil {
		mtf = b
		if in.CachedMTF != nil {
			*in.CachedMTF = mtf
		}
	}
	if in.Run != nil && mtf == nil {
		mtf = BuildKlinePatternMTFForDiagnoseRequest(true, phaseCtx, in.Run, in.Code, in.SignalFreq, in.Months, in.Bars)
		if in.CachedMTF != nil {
			*in.CachedMTF = mtf
		}
		StoreCachedLabKlineMTF(in.SessionID, in.RunID, mtf)
	}
	var patIn KlinePatternDisplayInput
	if in.Run != nil {
		patIn = NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, in.Bars, in.ProbeRaw, mtf)
	} else {
		patIn = NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, in.Bars, in.ProbeRaw, mtf)
		if mtf == nil {
			out.Degraded = append(out.Degraded, "kline_mtf_probe_unavailable")
		}
	}
	out.KlinePattern = buildLabKlineDetail(idx, patIn)
	applyLabKlinePatternTopLevel(&out)

	labFetcher := WrapCachedLabBarsFetcher(in.SessionID, in.RunID, wrapMTFFallbackBarsFetcher(in.LabFetcher, mtf))
	paCtx, paCancel := labDetailPAContext(cacheCtx, in.PATimeout)
	defer paCancel()
	if labFetcher != nil && strings.TrimSpace(in.Code) != "" {
		row := barRowAt(in.Bars, idx)
		dailyForAnchor := dailyBarsForLabAnchorFromMTF(mtf)
		if len(dailyForAnchor) == 0 {
			dailyForAnchor = dailyBarsForLabAnchor(phaseCtx, LabDetailInput{
				Run: in.Run, Code: in.Code, SignalFreq: in.SignalFreq, Months: in.Months,
				KeyLevels: in.KeyLevels, ProbeRaw: in.ProbeRaw, Bars: in.Bars, LookbackBars: in.LookbackBars,
				LabFetcher: labFetcher, SessionID: in.SessionID, RunID: in.RunID,
			})
		}
		anchorTime, _, anchorOK := anchorForSignalBar(row, dailyForAnchor)
		var asOf *time.Time
		if anchorOK {
			t := anchorTime
			asOf = &t
		}
		pa := priceanalysis.Run(paCtx, labFetcher, priceanalysis.RunRequest{
			Code:       in.Code,
			Months:     CapMonthsBack(in.Months),
			SignalFreq: in.SignalFreq,
			AsOf:       asOf,
			SpotPrice:  ent.Close,
			KeyLevels:  keyLevelsToLab(in.KeyLevels),
		})
		pa = priceanalysis.RoundPriceAnalysisForOutput(pa)
		converted := priceAnalysisFromLab(pa)
		out.PriceAnalysis = &converted
		jevCtx, jevCancel := labDetailJEVContext(cacheCtx, in.JEVTimeout)
		defer jevCancel()
		applyPriceAnalysisJEV(jevCtx, &out, side, in.Jev)
		if !anchorOK {
			out.Degraded = append(out.Degraded, "price_anchor_missing")
		}
	} else {
		out.Degraded = append(out.Degraded, "lab_price_fetch_unavailable")
	}
	ApplyLabJEVUnified(&out)
	return out, nil
}

func buildLabFlowDetail(ctx context.Context, in LabDetailInput, ref string) (LabEpisodeDetail, error) {
	out := LabEpisodeDetail{Ref: ref, Side: "flow", BarIndex: 0}
	if len(in.Bars) == 0 {
		return out, nil
	}
	idx := len(in.Bars) - 1
	ent := triggerRecordEntry(in.Bars, idx)
	out.Time = ent.Time
	out.Close = ent.Close

	cacheCtx := ContextWithPriceProbeCache(ctx)
	phaseCtx, phaseCancel := labDetailPhaseContext(cacheCtx, in.DetailTimeout)
	defer phaseCancel()
	if in.Run != nil {
		var mtf *KlinePatternMTFBundle
		if in.CachedMTF != nil && *in.CachedMTF != nil {
			mtf = *in.CachedMTF
		} else if b := CachedLabKlineMTF(in.SessionID, in.RunID); b != nil {
			mtf = b
			if in.CachedMTF != nil {
				*in.CachedMTF = mtf
			}
		} else {
			mtf = BuildKlinePatternMTFForDiagnoseRequest(true, phaseCtx, in.Run, in.Code, in.SignalFreq, in.Months, in.Bars)
			if in.CachedMTF != nil {
				*in.CachedMTF = mtf
			}
			StoreCachedLabKlineMTF(in.SessionID, in.RunID, mtf)
		}
		patIn := NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, in.Bars, in.ProbeRaw, mtf)
		out.KlinePattern = buildLabKlineDetail(idx, patIn)
		applyLabKlinePatternTopLevel(&out)
	}
	labFetcher := WrapCachedLabBarsFetcher(in.SessionID, in.RunID, in.LabFetcher)
	if labFetcher != nil && strings.TrimSpace(in.Code) != "" {
		pa := priceanalysis.Run(phaseCtx, labFetcher, priceanalysis.RunRequest{
			Code: in.Code, Months: CapMonthsBack(in.Months), SignalFreq: in.SignalFreq,
			SpotPrice: ent.Close, KeyLevels: keyLevelsToLab(in.KeyLevels),
		})
		pa = priceanalysis.RoundPriceAnalysisForOutput(pa)
		converted := priceAnalysisFromLab(pa)
		out.PriceAnalysis = &converted
		jevCtx, jevCancel := labDetailJEVContext(cacheCtx, in.JEVTimeout)
		defer jevCancel()
		applyPriceAnalysisJEV(jevCtx, &out, "flow", in.Jev)
	}
	ApplyLabJEVUnified(&out)
	return out, nil
}

func applyLabKlinePatternTopLevel(out *LabEpisodeDetail) {
	if out == nil || out.KlinePattern == nil {
		return
	}
	kp := out.KlinePattern
	out.KlinePatterns = kp.Patterns
	out.KlinePatternSummary = kp.Summary
	out.KlinePatternThreeBar = kp.ThreeBar
	out.KlinePatternBias = kp.Bias
	out.KlinePatternMTF = kp.MTF
}

func buildLabKlineDetail(idx int, in KlinePatternDisplayInput) *LabKlinePatternDetail {
	rec := EpisodeTriggerRecord{Index: idx}
	enrichTriggerKlineDisplayForDiagnose(true, &rec, in)
	detail := &LabKlinePatternDetail{
		Summary:  strings.TrimSpace(rec.KlinePatternSummary),
		Bias:     rec.KlinePatternBias,
		Patterns: rec.KlinePatterns,
		ThreeBar: rec.KlinePatternThreeBar,
		MTF:      rec.KlinePatternMTF,
	}
	if detail.MTF == nil && len(in.PatternHits) > 0 {
		detail.Patterns = kp.FilteredKlinePatternsAtIndex(in.PatternHits, idx)
		if detail.Summary == "" {
			detail.Summary = LightKlineSummaryAtIndex(in.PatternHits, idx)
		}
	}
	return detail
}

func dailyBarsForLabAnchor(ctx context.Context, in LabDetailInput) []any {
	if in.LabFetcher == nil {
		return nil
	}
	chart := coalesceLookback(in.LookbackBars, in.Bars)
	chartMonths := CapMonthsBack(in.Months)
	months := chartMonths
	var windowEnd time.Time
	if len(chart) > 0 {
		_, latest := barSeriesTimeExtent(chart)
		windowEnd = latest
		months = monthsBackForPerSignalPriceFetch(chartMonths, chart, "daily", windowEnd)
	}
	var asOf *time.Time
	if !windowEnd.IsZero() {
		end := windowEnd
		asOf = &end
	}
	bars, ok := in.LabFetcher.FetchBars(ctx, in.Code, "daily", months, asOf)
	if !ok {
		bars = nil
	}
	if len(bars) == 0 && NormalizeFrequency(in.SignalFreq) == "daily" && len(chart) > 0 {
		bars = chart
	}
	return bars
}

func coalesceLookback(lookback, bars []any) []any {
	if len(lookback) > 0 {
		return lookback
	}
	return bars
}

func keyLevelsToLab(kl KeyLevelBands) priceanalysis.KeyLevelBands {
	return priceanalysis.KeyLevelBands{
		SupportLow: kl.SupportLow, SupportHigh: kl.SupportHigh,
		ResistLow: kl.ResistLow, ResistHigh: kl.ResistHigh,
	}
}

func tfScoresFromLab(in map[string]priceanalysis.TFScoreBreakdown) map[string]TFScoreBreakdown {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]TFScoreBreakdown, len(in))
	for k, v := range in {
		out[k] = TFScoreBreakdown{
			Subscore: v.Subscore, Conclusion: v.Conclusion, Position: v.Position,
			BosChoch: v.BosChoch, Missing: v.Missing,
		}
	}
	return out
}

func priceAnalysisFromLab(pa priceanalysis.PriceAnalysis) PriceAnalysis {
	out := PriceAnalysis{
		Code: pa.Code, SpotPrice: pa.SpotPrice, SpotLabel: pa.SpotLabel,
		SupportResistance: pa.SupportResistance, Summary: pa.Summary,
		FilterTimeframe: pa.FilterTimeframe, FilterTimeframeNote: pa.FilterTimeframeNote,
		AnchorTime: pa.AnchorTime, AnchorClose: pa.AnchorClose,
		PriceScore: pa.PriceScore, TFScores: tfScoresFromLab(pa.TFScores), ScoreReason: pa.ScoreReason,
	}
	for _, row := range pa.Rows {
		out.Rows = append(out.Rows, PriceAnalysisRow{
			Timeframe: row.Timeframe, Conclusion: row.Conclusion, Structure: row.Structure,
			BosChoch: row.BosChoch, Position: row.Position, ER: row.ER, RVOL: row.RVOL,
			VolumeNote: row.VolumeNote,
		})
	}
	return out
}

type badLabRefError string

func (e badLabRefError) Error() string { return string(e) }

func errBadLabRef(msg string) error { return badLabRefError(msg) }

func labDetailPhaseContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func labDetailPAContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultLabDetailPATimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// labDetailJEVContext uses WithoutCancel so an expired phase deadline does not abort System One.
func labDetailJEVContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultLabDetailJEVTimeout
	}
	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}

// IsBadLabRef reports invalid ref / range errors for HTTP mapping.
func IsBadLabRef(err error) bool {
	_, ok := err.(badLabRefError)
	return ok
}
