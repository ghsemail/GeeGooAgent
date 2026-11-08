package strategydev

import (
	"context"
	"strings"
	"time"

	kp 	"github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const (
	// DefaultLabDetailPhaseTimeout bounds kline MTF probe for one lab-detail call.
	// Three upward probe_bot_signal_series calls (e.g. 60m+daily+weekly) run in parallel with up to
	// three retries each; 8s was too tight on cold market fetch and surfaced as MTF「拉数失败」+ price_anchor_missing.
	DefaultLabDetailPhaseTimeout = 30 * time.Second
	// DefaultLabDetailPATimeout is a separate budget for lab-api multi-TF price analysis (after MTF probe).
	DefaultLabDetailPATimeout = 45 * time.Second
	// DefaultLabDetailJEVTimeout is the dedicated JEV 决策 budget after PA / TA (matches JevClient HTTP timeout).
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
	// SignalEval when set, anchors detail.time to episode start_time on chart bars.
	SignalEval slots.SignalEpisodeEval
	LabFetcher    priceanalysis.BarsFetcher
	// DetailTimeout caps kline MTF probe only (not PA / JEV).
	DetailTimeout time.Duration
	// PATimeout caps lab-api price-analysis fetch (separate from MTF probe).
	PATimeout time.Duration
	// TATimeout caps lab-api technical-analysis fetch (daily + signal TF).
	TATimeout time.Duration
	// JEVTimeout is a separate budget for JEV 决策 after PA / TA completes.
	JEVTimeout time.Duration
	// CachedMTF when set, reuses MTF bundle across lab-detail calls for one diagnose run.
	CachedMTF **KlinePatternMTFBundle
	SessionID string
	RunID     string
	// Jev optional JEV client for lazy lab-detail decisions (PA + TA parallel paths).
	Jev *JevClient
	// TAClient optional lab-api client for POST /v1/lab/technical-analysis (same base URL/key as kline jobs).
	TAClient LabTechnicalAnalysisClient
	// IndexTrendClient optional lab-api client for POST /v1/lab/index-trend (same base URL/key as TA).
	IndexTrendClient LabIndexTrendClient
	// LongOnly matches signal_diagnose trade_direction (JEV copy + omitted short lab refs).
	LongOnly bool
	// KlineJEVProfile selects kline JEV facts (e.g. LabKlineJEVProfileDailyMacro for macro monitor).
	KlineJEVProfile string
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
	out := LabEpisodeDetail{Ref: ref, Side: side, BarIndex: idx, KlineJEVProfile: strings.TrimSpace(in.KlineJEVProfile)}
	chartBars := labDetailInputChartBars(in)
	if len(chartBars) == 0 || idx >= len(chartBars) {
		return out, errBadLabRef("bar index out of range: detail_ref suffix is chart start_idx (see lab_episode_summaries.start_idx), not episode list index")
	}
	chartIdx, anchorRow, anchorOK := resolveLabDetailChartBar(chartBars, in.SignalEval, side, idx)
	if !anchorOK || anchorRow == nil {
		return out, errBadLabRef("bar index out of range: detail_ref suffix is chart start_idx (see lab_episode_summaries.start_idx), not episode list index")
	}
	ent := triggerRecordEntry(chartBars, chartIdx)
	out.Time = ent.Time
	out.Close = ent.Close

	cacheCtx := ContextWithPriceProbeCache(ctx)
	phaseCtx, phaseCancel := labDetailPhaseContext(cacheCtx, in.DetailTimeout)
	defer phaseCancel()
	paCtx, paCancel := labDetailPAContext(cacheCtx, in.PATimeout)
	defer paCancel()

	if in.Run != nil && in.ProbeRaw != nil {
		kp.EnsurePatternHitsOnProbeRaw(phaseCtx, kp.ToolRunner(in.Run), in.Code, in.SignalFreq, in.Months, chartBars, in.ProbeRaw)
	}

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
		mtf = BuildKlinePatternMTFForDiagnoseRequest(true, phaseCtx, in.Run, in.Code, in.SignalFreq, in.Months, chartBars)
		if in.CachedMTF != nil {
			*in.CachedMTF = mtf
		}
		StoreCachedLabKlineMTF(in.SessionID, in.RunID, mtf)
	}
	var patIn KlinePatternDisplayInput
	if in.Run != nil {
		patIn = NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, chartBars, in.ProbeRaw, mtf)
	} else {
		patIn = NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, chartBars, in.ProbeRaw, mtf)
		if mtf == nil {
			out.Degraded = append(out.Degraded, "kline_mtf_probe_unavailable")
		}
	}
	out.KlinePattern = buildLabKlineDetail(chartIdx, patIn)
	applyLabKlinePatternTopLevel(&out)

	labFetcher := WrapCachedLabBarsFetcher(in.SessionID, in.RunID, wrapMTFFallbackBarsFetcher(in.LabFetcher, mtf))
	if labFetcher != nil && strings.TrimSpace(in.Code) != "" {
		row := anchorRow
		dailyForAnchor := dailyBarsForLabAnchorFromMTF(mtf)
		if len(dailyForAnchor) == 0 {
			dailyForAnchor = dailyBarsForLabAnchor(paCtx, LabDetailInput{
				Run: in.Run, Code: in.Code, SignalFreq: in.SignalFreq, Months: in.Months,
				KeyLevels: in.KeyLevels, ProbeRaw: in.ProbeRaw, Bars: chartBars, LookbackBars: in.LookbackBars,
				LabFetcher: labFetcher, SessionID: in.SessionID, RunID: in.RunID,
			})
		}
		priceAnchorTime, _, priceAnchorOK := anchorForSignalBar(row, dailyForAnchor)
		var asOf *time.Time
		if priceAnchorOK {
			t := priceAnchorTime
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
		var ta *LabTechnicalAnalysis
		var indexTrend *LabIndexTrend
		if in.TAClient != nil || in.IndexTrendClient != nil {
			taCtx, taCancel := labDetailTAContext(cacheCtx, in.TATimeout)
			if in.TAClient != nil {
				ta = fetchLabTechnicalAnalysis(taCtx, in.TAClient, in, asOf)
				out.TechnicalAnalysis = ta
			}
			if in.IndexTrendClient != nil {
				indexTrend = fetchLabIndexTrend(taCtx, in.IndexTrendClient, in, asOf)
				out.IndexTrend = indexTrend
			}
			taCancel()
		}
		jevCtx, jevCancel := labDetailJEVContext(cacheCtx, in.JEVTimeout)
		defer jevCancel()
		applyPriceAnalysisJEV(jevCtx, &out, side, in.Jev, in.LongOnly)
		applyKlineJEV(jevCtx, &out, side, in.Jev, in.LongOnly)
		if ta != nil {
			rec := ResolveTechnicalAnalysisJEV(jevCtx, in.Jev, ta, side, in.SignalFreq, in.LongOnly)
			out.TechnicalAnalysisJEV = &rec
		}
		if indexTrend != nil {
			rec := ResolveIndexTrendJEV(jevCtx, in.Jev, indexTrend, side, in.SignalFreq, in.LongOnly)
			out.IndexTrendJEV = &rec
		}
		applyLabJEVUnified(jevCtx, &out, side, in)
		if !priceAnchorOK {
			out.Degraded = append(out.Degraded, "price_anchor_missing")
		}
	} else {
		out.Degraded = append(out.Degraded, "lab_price_fetch_unavailable")
	}
	PrepareLabEpisodeDetailForEmit(&out)
	return out, nil
}

func buildLabFlowDetail(ctx context.Context, in LabDetailInput, ref string) (LabEpisodeDetail, error) {
	out := LabEpisodeDetail{Ref: ref, Side: "flow", BarIndex: 0, KlineJEVProfile: strings.TrimSpace(in.KlineJEVProfile)}
	chartBars := labDetailInputChartBars(in)
	if len(chartBars) == 0 {
		return out, nil
	}
	idx := len(chartBars) - 1
	ent := triggerRecordEntry(chartBars, idx)
	out.Time = ent.Time
	out.Close = ent.Close

	cacheCtx := ContextWithPriceProbeCache(ctx)
	phaseCtx, phaseCancel := labDetailPhaseContext(cacheCtx, in.DetailTimeout)
	defer phaseCancel()
	if in.Run != nil && in.ProbeRaw != nil {
		kp.EnsurePatternHitsOnProbeRaw(phaseCtx, kp.ToolRunner(in.Run), in.Code, in.SignalFreq, in.Months, chartBars, in.ProbeRaw)
	}
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
			mtf = BuildKlinePatternMTFForDiagnoseRequest(true, phaseCtx, in.Run, in.Code, in.SignalFreq, in.Months, chartBars)
			if in.CachedMTF != nil {
				*in.CachedMTF = mtf
			}
			StoreCachedLabKlineMTF(in.SessionID, in.RunID, mtf)
		}
		patIn := NewKlinePatternDisplayInputForDiagnose(true, in.SignalFreq, chartBars, in.ProbeRaw, mtf)
		out.KlinePattern = buildLabKlineDetail(idx, patIn)
		applyLabKlinePatternTopLevel(&out)
	}
	labFetcher := WrapCachedLabBarsFetcher(in.SessionID, in.RunID, in.LabFetcher)
	paCtx, paCancel := labDetailPAContext(cacheCtx, in.PATimeout)
	defer paCancel()
	if labFetcher != nil && strings.TrimSpace(in.Code) != "" {
		pa := priceanalysis.Run(paCtx, labFetcher, priceanalysis.RunRequest{
			Code: in.Code, Months: CapMonthsBack(in.Months), SignalFreq: in.SignalFreq,
			SpotPrice: ent.Close, KeyLevels: keyLevelsToLab(in.KeyLevels),
		})
		pa = priceanalysis.RoundPriceAnalysisForOutput(pa)
		converted := priceAnalysisFromLab(pa)
		out.PriceAnalysis = &converted
		var ta *LabTechnicalAnalysis
		var indexTrend *LabIndexTrend
		if in.TAClient != nil || in.IndexTrendClient != nil {
			taCtx, taCancel := labDetailTAContext(cacheCtx, in.TATimeout)
			if in.TAClient != nil {
				ta = fetchLabTechnicalAnalysis(taCtx, in.TAClient, in, nil)
				out.TechnicalAnalysis = ta
			}
			if in.IndexTrendClient != nil {
				indexTrend = fetchLabIndexTrend(taCtx, in.IndexTrendClient, in, nil)
				out.IndexTrend = indexTrend
			}
			taCancel()
		}
		jevCtx, jevCancel := labDetailJEVContext(cacheCtx, in.JEVTimeout)
		defer jevCancel()
		applyPriceAnalysisJEV(jevCtx, &out, "flow", in.Jev, in.LongOnly)
		applyKlineJEV(jevCtx, &out, "flow", in.Jev, in.LongOnly)
		if ta != nil {
			rec := ResolveTechnicalAnalysisJEV(jevCtx, in.Jev, ta, "flow", in.SignalFreq, in.LongOnly)
			out.TechnicalAnalysisJEV = &rec
		}
		if indexTrend != nil {
			rec := ResolveIndexTrendJEV(jevCtx, in.Jev, indexTrend, "flow", in.SignalFreq, in.LongOnly)
			out.IndexTrendJEV = &rec
		}
		applyLabJEVUnified(jevCtx, &out, "flow", in)
	}
	PrepareLabEpisodeDetailForEmit(&out)
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

// labDetailJEVContext uses WithoutCancel so an expired phase deadline does not abort JEV 决策.
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
