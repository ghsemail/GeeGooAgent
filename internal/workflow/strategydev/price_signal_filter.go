package strategydev

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// PerSignalPriceFilterInput enables anchored multi-TF analysis per trigger.
type PerSignalPriceFilterInput struct {
	Run    ToolRunner
	Code   string
	Months int
	KL     KeyLevelBands
	// AnnotateOnly when true computes per-trigger price analysis and would-intercept labels
	// without zeroing buy_merged/sell_merged (soft gate, default product mode).
	AnnotateOnly bool
	// ScoreThreshold for hard gate (default 0); buys blocked when score < threshold.
	ScoreThreshold *float64
	// LookbackChartBars is the pre-clamp probe span used for multi-TF fetch sizing and signal-TF OHLCV.
	LookbackChartBars []any
}

// ApplyPerSignalPriceAnalysisEpisodeFilter masks triggers using each signal's anchored multi-TF gate.
func ApplyPerSignalPriceAnalysisEpisodeFilter(
	ctx context.Context,
	in PerSignalPriceFilterInput,
	bars []any,
	freq string,
	buyMerged, sellMerged []any,
) ([]any, []any, PriceFilterResult, EpisodeTriggerFilterStats) {
	gateFreq := NormalizeFrequency(freq)
	scoreThreshold := PriceAnalysisScoreThreshold(in.ScoreThreshold)
	basis := fmt.Sprintf("按各触发前一交易日锚点多周期打分，综合分低于 %.0f 时掩码剔除买入；卖出在综合分高于 %.0f 时掩码", scoreThreshold, -scoreThreshold)
	if in.AnnotateOnly {
		basis = "软门控：逐条多周期打分并标注本会拦截的触发，不掩码剔除（Episode 评价与未门控一致）"
	}
	stats := EpisodeTriggerFilterStats{
		InterceptionBasis: basis,
	}
	if len(buyMerged) == 0 && len(sellMerged) == 0 {
		return buyMerged, sellMerged, AnalyzePriceFilter(bars, gateFreq), stats
	}
	stats.PreFilterBuyHits = countMergedSignal(buyMerged, 1)
	stats.PreFilterSellHits = countMergedSignal(sellMerged, -1)

	chartMonths := CapMonthsBack(in.Months)
	_, chartLatest := barSeriesTimeExtent(bars)
	var windowEnd *time.Time
	if !chartLatest.IsZero() {
		end := chartLatest
		windowEnd = &end
	}
	lookbackBars := in.LookbackChartBars
	if len(lookbackBars) == 0 {
		lookbackBars = bars
	}
	// One probe per TF (as_of = diagnosis window end when known); per-signal anchor truncate is local below.
	barsByFreq, fetchOKByFreq := FetchMultiTimeframePriceAnalysisBars(ctx, MultiTimeframePriceBarsInput{
		Run: in.Run, Code: in.Code, Months: chartMonths, GateFreq: gateFreq, AsOf: windowEnd, ChartBars: lookbackBars,
	})
	dailyBars := barsByFreq["daily"]
	if len(dailyBars) == 0 && gateFreq == "daily" {
		dailyBars = bars
	}

	type triggerJob struct {
		idx   int
		side  string // buy | sell
		close float64
		time  string
	}
	var jobs []triggerJob
	n := len(buyMerged)
	if len(sellMerged) > n {
		n = len(sellMerged)
	}
	for i := 0; i < n; i++ {
		if i < len(buyMerged) && int(slots.FloatAny(buyMerged[i])) == 1 {
			ent := triggerLedgerEntry(bars, i)
			jobs = append(jobs, triggerJob{idx: i, side: "buy", close: ent.Close, time: ent.Time})
		}
		if i < len(sellMerged) && int(slots.FloatAny(sellMerged[i])) == -1 {
			ent := triggerLedgerEntry(bars, i)
			jobs = append(jobs, triggerJob{idx: i, side: "sell", close: ent.Close, time: ent.Time})
		}
	}

	maskBuy := map[int]bool{}
	maskSell := map[int]bool{}
	recordByIdxSide := map[string]EpisodeTriggerRecord{}

	var lastGate PriceFilterResult
	for _, job := range jobs {
		row := barRowAt(bars, job.idx)
		anchorTime, anchorClose, anchorOK := anchorForSignalBar(row, dailyBars)
		var pa PriceAnalysis
		var pfGate PriceFilterResult
		if !anchorOK {
			pa = buildPriceAnalysisWhenPriorTradingDayMissing(job.close, gateFreq, in.KL)
			pfGate = episodeGateWhenPriorTradingDayMissing(gateFreq)
		} else {
			pa = buildAnchoredPriceAnalysis(barsByFreq, fetchOKByFreq, lookbackBars, gateFreq, anchorTime, anchorClose, job.close, in.KL)
			pfGate = priceFilterForGateFrequency(pa, gateFreq, barsByFreq, lookbackBars, anchorTime, gateFreq)
		}
		scoreResult := attachScoreToPriceAnalysis(&pa, gateFreq)
		lastGate = pfGate
		wouldIntercept := ShouldInterceptTriggerByScore(job.side, scoreResult.Score, scoreThreshold)
		hardMask := !in.AnnotateOnly
		rec := EpisodeTriggerRecord{
			Index:                 job.idx,
			Time:                  job.time,
			Close:                 job.close,
			WouldIntercept:        wouldIntercept,
			Intercepted:           hardMask && wouldIntercept,
			PriceAnalysis:         &pa,
			PriceScore:            scoreResult.Score,
			TFScores:              scoreResult.TFScores,
			ScoreReason:           scoreResult.ScoreReason,
			AIInterceptConclusion: RuleDerivedInterceptConclusionScore(job.side, wouldIntercept, hardMask, pa, scoreResult, scoreThreshold),
			AIInterceptConclusionSource: "rule",
		}
		if hardMask && wouldIntercept {
			rec.Masked = true
		}
		key := fmt.Sprintf("%s:%d", job.side, job.idx)
		recordByIdxSide[key] = rec
		if hardMask {
			if job.side == "buy" {
				if wouldIntercept {
					maskBuy[job.idx] = true
				}
			} else if wouldIntercept {
				maskSell[job.idx] = true
			}
		} else if wouldIntercept {
			if job.side == "buy" {
				stats.WouldInterceptBuy++
			} else {
				stats.WouldInterceptSell++
			}
		}
	}

	outBuy := cloneMerged(buyMerged, n)
	outSell := cloneMerged(sellMerged, n)
	if !in.AnnotateOnly {
		for i := 0; i < n; i++ {
			if maskBuy[i] {
				outBuy[i] = 0
				stats.StrippedBuy++
			}
			if maskSell[i] {
				outSell[i] = 0
				stats.StrippedSell++
			}
		}
	}

	for _, job := range jobs {
		key := fmt.Sprintf("%s:%d", job.side, job.idx)
		rec := recordByIdxSide[key]
		if job.side == "buy" {
			stats.BuyTriggers = append(stats.BuyTriggers, rec)
			if rec.WouldIntercept {
				stats.StrippedBuyTriggers = append(stats.StrippedBuyTriggers, rec)
			} else {
				stats.KeptBuyTriggers = append(stats.KeptBuyTriggers, rec)
			}
		} else {
			stats.SellTriggers = append(stats.SellTriggers, rec)
			if rec.WouldIntercept {
				stats.StrippedSellTriggers = append(stats.StrippedSellTriggers, rec)
			} else {
				stats.KeptSellTriggers = append(stats.KeptSellTriggers, rec)
			}
		}
	}

	if in.AnnotateOnly {
		stats.PostFilterBuyHits = stats.PreFilterBuyHits
		stats.PostFilterSellHits = stats.PreFilterSellHits
	} else {
		stats.PostFilterBuyHits = countMergedSignal(outBuy, 1)
		stats.PostFilterSellHits = countMergedSignal(outSell, -1)
	}
	runPF := AnalyzePriceFilter(bars, gateFreq)
	if lastGate.BarCount > 0 {
		runPF = lastGate
	}
	return outBuy, outSell, runPF, stats
}

func cloneMerged(src []any, n int) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		if i < len(src) {
			out[i] = src[i]
		} else {
			out[i] = 0
		}
	}
	return out
}

func barRowAt(bars []any, idx int) map[string]any {
	if idx < 0 || idx >= len(bars) {
		return nil
	}
	row, _ := bars[idx].(map[string]any)
	return row
}

func anchorForSignalBar(signalRow map[string]any, dailyBars []any) (anchorTime time.Time, anchorClose float64, ok bool) {
	if signalRow == nil || len(dailyBars) == 0 {
		return time.Time{}, 0, false
	}
	signalDay := calendarDateAsiaShanghai(barTimeUnix(signalRow))
	if signalDay.IsZero() {
		return time.Time{}, 0, false
	}
	var anchorRow map[string]any
	var anchorDay time.Time
	for _, raw := range dailyBars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		d := calendarDateAsiaShanghai(barTimeUnix(row))
		if d.IsZero() || !d.Before(signalDay) {
			continue
		}
		if anchorRow == nil || d.After(anchorDay) {
			anchorRow = row
			anchorDay = d
		}
	}
	if anchorRow == nil {
		return time.Time{}, 0, false
	}
	anchorTime = anchorInstantFromDailyBar(anchorRow)
	anchorClose = slots.FloatAny(anchorRow["close"])
	return anchorTime, anchorClose, !anchorTime.IsZero()
}

func calendarDateAsiaShanghai(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	d := ts.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

func anchorInstantFromDailyBar(row map[string]any) time.Time {
	ts := barTimeUnix(row)
	if ts.IsZero() {
		return time.Time{}
	}
	cal := calendarDateAsiaShanghai(ts)
	if cal.IsZero() {
		return time.Time{}
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return time.Date(cal.Year(), cal.Month(), cal.Day(), 16, 0, 0, 0, loc)
}

func buildPriceAnalysisWhenPriorTradingDayMissing(signalClose float64, chartFreq string, kl KeyLevelBands) PriceAnalysis {
	pa := PriceAnalysis{
		SpotPrice:       signalClose,
		FilterTimeframe: timeframeDisplayLabel(chartFreq),
		SpotLabel:       "前一交易日锚点不可用",
	}
	for _, spec := range multiTimeframeSpecsForGate(chartFreq) {
		pa.Rows = append(pa.Rows, emptyPriceAnalysisRow(spec.Label, structurePriorTradingDayUnavailable))
	}
	pa.Summary = "前一交易日日线锚点不可用，Episode 多周期打分仅供参考。"
	pa.FilterTimeframeNote = fmt.Sprintf("Episode 方向门控使用 %s 周期结论（锚点前）", timeframeDisplayLabel(chartFreq))
	pa.SupportResistance = formatSupportResistanceLine(kl, pa.Rows)
	return pa
}

func episodeGateWhenPriorTradingDayMissing(gateFreq string) PriceFilterResult {
	return PriceFilterResult{
		Regime:     RegimeUnclear,
		Frequency:  NormalizeFrequency(gateFreq),
		AllowLong:  true,
		AllowShort: true,
		Notes:      []string{"前一交易日锚点不可用，跳过 Episode 价格门控"},
	}
}

func buildAnchoredPriceAnalysis(
	barsByFreq map[string][]any,
	fetchOKByFreq map[string]bool,
	chartBars []any,
	chartFreq string,
	anchorTime time.Time,
	anchorClose, signalClose float64,
	kl KeyLevelBands,
) PriceAnalysis {
	pa := PriceAnalysis{
		SpotPrice:       signalClose,
		FilterTimeframe: timeframeDisplayLabel(chartFreq),
	}
	if anchorClose > 0 {
		pa.SpotLabel = fmt.Sprintf("锚点收盘 %.2g", anchorClose)
	}
	if !anchorTime.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		pa.AnchorTime = anchorTime.In(loc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
	}
	pa.AnchorClose = anchorClose

	for _, spec := range multiTimeframeSpecsForGate(chartFreq) {
		src := ohlcvSourceForTimeframe(spec.Freq, chartFreq, barsByFreq, chartBars, anchorTime)
		trunc := truncateBarsAtAnchor(src, anchorTime)
		if len(trunc) == 0 {
			reason := structureReasonAfterAnchorTruncate(len(src), fetchOKByFreq[spec.Freq])
			pa.Rows = append(pa.Rows, emptyPriceAnalysisRow(spec.Label, reason))
			continue
		}
		analysisBars := tailBarsForPriceAnalysis(trunc, spec.Freq)
		pa.Rows = append(pa.Rows, BuildPriceAnalysisRow(analysisBars, spec.Label, spec.Freq))
	}
	pa.SupportResistance = formatSupportResistanceLine(kl, pa.Rows)
	pa.Summary = SummarizeMultiTimeframeRows(pa.Rows)
	pa.FilterTimeframeNote = fmt.Sprintf("Episode 方向门控使用 %s 周期结论（锚点前）", timeframeDisplayLabel(chartFreq))
	return pa
}

func truncateBarsAtAnchor(bars []any, anchorTime time.Time) []any {
	if len(bars) == 0 {
		return nil
	}
	if anchorTime.IsZero() {
		return nil
	}
	out := make([]any, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() || !ts.After(anchorTime) {
			out = append(out, raw)
		}
	}
	return out
}

func priceFilterForGateFrequency(
	pa PriceAnalysis,
	gateFreq string,
	barsByFreq map[string][]any,
	chartBars []any,
	anchorTime time.Time,
	chartFreq string,
) PriceFilterResult {
	gateFreq = NormalizeFrequency(gateFreq)
	participating := hardGateParticipatingFrequencies(gateFreq)
	var gatePF PriceFilterResult
	allowLong := true
	for _, freq := range participating {
		pf := priceFilterResultForFrequency(pa, freq, barsByFreq, chartBars, anchorTime, chartFreq)
		if freq == gateFreq {
			gatePF = pf
		}
		if !pf.AllowLong {
			allowLong = false
		}
	}
	if gatePF.Frequency == "" {
		gatePF = priceFilterResultForFrequency(pa, gateFreq, barsByFreq, chartBars, anchorTime, chartFreq)
	}
	gatePF.AllowLong = allowLong
	// Reporting: shorts disallowed only on gate uptrend (intercept uses Regime directly).
	gatePF.AllowShort = gatePF.Regime != RegimeUp
	return gatePF
}

// RuleDerivedInterceptConclusionScore summarizes intercept from multi-TF score.
func RuleDerivedInterceptConclusionScore(side string, wouldIntercept, hardMask bool, pa PriceAnalysis, sc MultiTFPriceScore, threshold float64) string {
	dir := "买入"
	if side == "sell" {
		dir = "卖出"
	}
	reason := strings.TrimSpace(sc.ScoreReason)
	if reason == "" {
		reason = fmt.Sprintf("综合分 %.0f", sc.Score)
	}
	anchorMissing := strings.Contains(pa.SpotLabel, structurePriorTradingDayUnavailable) ||
		strings.Contains(pa.Summary, "锚点不可用")
	if wouldIntercept {
		verb := "已拦截"
		if !hardMask {
			verb = "本会拦截"
		}
		limit := fmt.Sprintf("%.0f", threshold)
		if side == "sell" {
			limit = fmt.Sprintf("%.0f", -threshold)
		}
		return fmt.Sprintf("%s%s信号：综合分 %.0f 未达阈值 %s；%s", verb, dir, sc.Score, limit, reason)
	}
	if anchorMissing && !hardMask {
		return fmt.Sprintf("保留%s信号（软门控）：前一交易日锚点不可用，综合分 %.0f；%s", dir, sc.Score, reason)
	}
	if anchorMissing && hardMask {
		return fmt.Sprintf("通过%s信号：锚点不可用但综合分 %.0f 未触发硬拦；%s", dir, sc.Score, reason)
	}
	return fmt.Sprintf("通过%s信号：综合分 %.0f；%s", dir, sc.Score, reason)
}

// RuleDerivedInterceptConclusion is the legacy regime-based summary (non per-episode paths).
func RuleDerivedInterceptConclusion(side string, wouldIntercept, hardMask bool, pa PriceAnalysis, pf PriceFilterResult) string {
	dir := "买入"
	if side == "sell" {
		dir = "卖出"
	}
	if wouldIntercept {
		reason := strings.TrimSpace(BuildPriceFilterInterceptionBasis(pf))
		if reason == "" {
			reason = fmt.Sprintf("%s 周期结构为 %s", timeframeDisplayLabel(pf.Frequency), pf.Regime)
		}
		verb := "已拦截"
		if !hardMask {
			verb = "本会拦截"
		}
		if s := strings.TrimSpace(pa.Summary); s != "" {
			return fmt.Sprintf("%s%s信号：%s；%s", verb, dir, reason, s)
		}
		return fmt.Sprintf("%s%s信号：%s", verb, dir, reason)
	}
	if s := strings.TrimSpace(pa.Summary); s != "" {
		return fmt.Sprintf("通过%s信号：%s 周期允许该方向；%s", dir, timeframeDisplayLabel(pf.Frequency), s)
	}
	return fmt.Sprintf("通过%s信号：%s 周期允许该方向", dir, timeframeDisplayLabel(pf.Frequency))
}
