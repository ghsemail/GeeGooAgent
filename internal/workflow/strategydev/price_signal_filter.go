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
	stats := EpisodeTriggerFilterStats{
		InterceptionBasis: "按各触发信号日历日的前一交易日收盘为锚点，逐条多周期价格分析后掩码剔除",
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
	// One probe per TF (as_of = diagnosis window end when known); per-signal anchor truncate is local below.
	barsByFreq, fetchOKByFreq := FetchMultiTimeframePriceAnalysisBars(ctx, MultiTimeframePriceBarsInput{
		Run: in.Run, Code: in.Code, Months: chartMonths, GateFreq: gateFreq, AsOf: windowEnd, ChartBars: bars,
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
			pa = buildAnchoredPriceAnalysis(barsByFreq, fetchOKByFreq, bars, gateFreq, anchorTime, anchorClose, job.close, in.KL)
			pfGate = priceFilterForGateFrequency(pa, gateFreq, barsByFreq, bars, anchorTime, gateFreq)
		}
		lastGate = pfGate
		intercept := shouldInterceptTrigger(job.side, pfGate)
		rec := EpisodeTriggerRecord{
			Index:                 job.idx,
			Time:                  job.time,
			Close:                 job.close,
			Masked:                intercept,
			Intercepted:           intercept,
			PriceAnalysis:         &pa,
			AIInterceptConclusion: RuleDerivedInterceptConclusion(job.side, intercept, pa, pfGate),
			AIInterceptConclusionSource: "rule",
		}
		key := fmt.Sprintf("%s:%d", job.side, job.idx)
		recordByIdxSide[key] = rec
		if job.side == "buy" {
			if intercept {
				maskBuy[job.idx] = true
			}
		} else if intercept {
			maskSell[job.idx] = true
		}
	}

	outBuy := cloneMerged(buyMerged, n)
	outSell := cloneMerged(sellMerged, n)
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

	for _, job := range jobs {
		key := fmt.Sprintf("%s:%d", job.side, job.idx)
		rec := recordByIdxSide[key]
		if job.side == "buy" {
			stats.BuyTriggers = append(stats.BuyTriggers, rec)
			if rec.Masked {
				stats.StrippedBuyTriggers = append(stats.StrippedBuyTriggers, rec)
			} else {
				stats.KeptBuyTriggers = append(stats.KeptBuyTriggers, rec)
			}
		} else {
			stats.SellTriggers = append(stats.SellTriggers, rec)
			if rec.Masked {
				stats.StrippedSellTriggers = append(stats.StrippedSellTriggers, rec)
			} else {
				stats.KeptSellTriggers = append(stats.KeptSellTriggers, rec)
			}
		}
	}

	stats.PostFilterBuyHits = countMergedSignal(outBuy, 1)
	stats.PostFilterSellHits = countMergedSignal(outSell, -1)
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
	pa.Summary = "前一交易日日线锚点不可用，未做多周期结构门控。"
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
		src := barsByFreq[spec.Freq]
		if len(src) == 0 && NormalizeFrequency(chartFreq) == spec.Freq {
			src = chartBars
		}
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

// RuleDerivedInterceptConclusion is the fallback one-line intercept summary.
func RuleDerivedInterceptConclusion(side string, intercepted bool, pa PriceAnalysis, pf PriceFilterResult) string {
	dir := "买入"
	if side == "sell" {
		dir = "卖出"
	}
	if intercepted {
		reason := strings.TrimSpace(BuildPriceFilterInterceptionBasis(pf))
		if reason == "" {
			reason = fmt.Sprintf("%s 周期结构为 %s", timeframeDisplayLabel(pf.Frequency), pf.Regime)
		}
		if s := strings.TrimSpace(pa.Summary); s != "" {
			return fmt.Sprintf("已拦截%s信号：%s；%s", dir, reason, s)
		}
		return fmt.Sprintf("已拦截%s信号：%s", dir, reason)
	}
	if s := strings.TrimSpace(pa.Summary); s != "" {
		return fmt.Sprintf("通过%s信号：%s 周期允许该方向；%s", dir, timeframeDisplayLabel(pf.Frequency), s)
	}
	return fmt.Sprintf("通过%s信号：%s 周期允许该方向", dir, timeframeDisplayLabel(pf.Frequency))
}
