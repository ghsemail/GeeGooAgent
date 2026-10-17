package strategydev

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const minIndicatorWarmupBars = 100

// IndicatorWarmupMeta describes indicator warmup applied to a diagnose probe payload.
type IndicatorWarmupMeta struct {
	RequiredWarmupBars     int    `json:"required_warmup_bars,omitempty"`
	PrefixBarsBeforeWindow int    `json:"prefix_bars_before_window,omitempty"`
	SuppressedInWindowBars int    `json:"suppressed_in_window_bars,omitempty"`
	Note                   string `json:"note,omitempty"`
}

// indicatorWarmupBarCount returns bars to prepend before the analysis window for indicator convergence.
func indicatorWarmupBarCount(buySignal, sellSignal []any) int {
	slowest := slowestIndicatorPeriod(buySignal)
	if s := slowestIndicatorPeriod(sellSignal); s > slowest {
		slowest = s
	}
	if slowest < 1 {
		slowest = 26 // MACD default slow period
	}
	w := slowest * 2
	if w < minIndicatorWarmupBars {
		w = minIndicatorWarmupBars
	}
	return w
}

func slowestIndicatorPeriod(chain []any) int {
	maxP := 0
	for _, raw := range chain {
		rule, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if p := indicatorPeriodFromRule(rule); p > maxP {
			maxP = p
		}
	}
	return maxP
}

func indicatorPeriodFromRule(rule map[string]any) int {
	idx := strings.ToUpper(strings.TrimSpace(fmt.Sprint(rule["index"])))
	param, _ := rule["param"].(map[string]any)
	switch idx {
	case "MACD":
		if p := intParam(param, "slowperiod", "slow_period", "slowPeriod", "longperiod", "long_period"); p > 0 {
			return p
		}
		return 26
	case "EMA", "SMA", "MA":
		if p := intParam(param, "period", "timeperiod", "time_period", "length"); p > 0 {
			return p
		}
		return 20
	case "RSI":
		if p := intParam(param, "period", "timeperiod", "time_period"); p > 0 {
			return p
		}
		return 14
	case "SAR":
		return 20
	case "ADX", "ATR", "BBAND", "BOLL", "KDJ":
		if p := intParam(param, "period", "timeperiod", "time_period"); p > 0 {
			return p
		}
		return 14
	default:
		if p := intParam(param, "period", "timeperiod", "slowperiod", "slow_period"); p > 0 {
			return p
		}
	}
	return 0
}

func intParam(param map[string]any, keys ...string) int {
	if param == nil {
		return 0
	}
	for _, k := range keys {
		if v, ok := param[k]; ok {
			n := int(slots.FloatAny(v))
			if n > 0 {
				return n
			}
		}
	}
	return 0
}

// EnrichProbeArgsForIndicatorWarmup extends probe fetch span so GeeGooSignal can compute
// indicators on a warmup prefix before the capped analysis window. Returns required warmup bars.
func EnrichProbeArgsForIndicatorWarmup(args map[string]any, buySignal, sellSignal []any) int {
	if args == nil {
		return 0
	}
	warmup := indicatorWarmupBarCount(buySignal, sellSignal)
	if warmup <= 0 {
		return 0
	}
	months := CapMonthsBack(intFromAnyArg(args["months_back"]))
	freq, _ := args["frequency"].(string)
	windowEnd := parseDiagnoseTimeString(strings.TrimSpace(fmt.Sprint(args["as_of"])))
	if windowEnd.IsZero() {
		windowEnd = parseDiagnoseTimeString(strings.TrimSpace(fmt.Sprint(args["end"])))
	}
	extMonths := monthsBackForIndicatorWarmupFetch(months, freq, warmup, windowEnd)
	if extMonths > months {
		args["months_back"] = extMonths
	}
	if limit := probeFetchLimitWithWarmup(months, freq, warmup); limit > 0 {
		if cur := intFromAnyArg(args["limit"]); limit > cur {
			args["limit"] = limit
		}
	}
	return warmup
}

func intFromAnyArg(v any) int {
	if v == nil {
		return 0
	}
	return int(slots.FloatAny(v))
}

func monthsBackForIndicatorWarmupFetch(chartMonths int, freq string, warmupBars int, windowEnd time.Time) int {
	base := CapMonthsBack(chartMonths)
	if warmupBars <= 0 {
		return base
	}
	if windowEnd.IsZero() {
		windowEnd = time.Now()
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	windowStart := windowEnd.In(loc).AddDate(0, -base, 0)
	needStart := windowStart.Add(-warmupBarCalendarLookback(freq, warmupBars))
	spanMonths := ceilCalendarMonthsBetween(needStart, windowEnd.In(loc))
	if spanMonths > base {
		base = spanMonths
	}
	return capMonthsBackForPerSignalPriceFetch(freq, base)
}

func warmupBarCalendarLookback(freq string, warmupBars int) time.Duration {
	if warmupBars <= 0 {
		return 0
	}
	switch NormalizeFrequency(freq) {
	case "5m":
		days := (warmupBars / 48) + 3
		return time.Duration(days) * 24 * time.Hour
	case "15m":
		days := (warmupBars / 16) + 3
		return time.Duration(days) * 24 * time.Hour
	case "60m":
		days := (warmupBars / 4) + 5
		return time.Duration(days) * 24 * time.Hour
	case "daily":
		days := warmupBars + 10
		return time.Duration(days) * 24 * time.Hour
	case "weekly", "1wk":
		weeks := warmupBars + 4
		return time.Duration(weeks*7) * 24 * time.Hour
	default:
		days := (warmupBars / 4) + 5
		return time.Duration(days) * 24 * time.Hour
	}
}

func probeFetchLimitWithWarmup(analysisMonths int, freq string, warmupBars int) int {
	total := estimatedAnalysisBarCount(analysisMonths, freq) + warmupBars
	if total > 800 {
		total = 800
	}
	if total < warmupBars+1 {
		total = warmupBars + 1
	}
	return total
}

func estimatedAnalysisBarCount(months int, freq string) int {
	per := analysisBarsLimit(freq)
	m := CapMonthsBack(months)
	if m <= 0 {
		m = 3
	}
	return per * m
}

func preAnalysisWindowBarCount(data map[string]any, analysisMonths int) int {
	bars, _ := data["bars"].([]any)
	if len(bars) == 0 {
		return 0
	}
	windowEnd := diagnoseWindowEnd(data, bars)
	if windowEnd.IsZero() {
		return 0
	}
	months := CapMonthsBack(analysisMonths)
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	windowStart := windowEnd.In(loc).AddDate(0, -months, 0)
	startIdx, _ := barIndexRangeForWindow(bars, windowStart, windowEnd)
	return startIdx
}

// FinalizeDiagnoseProbeSeries clamps to analysisMonths and suppresses signals on leading
// in-window bars when warmup history was shorter than required.
func FinalizeDiagnoseProbeSeries(data map[string]any, analysisMonths int, buySignal, sellSignal []any) IndicatorWarmupMeta {
	meta := IndicatorWarmupMeta{RequiredWarmupBars: indicatorWarmupBarCount(buySignal, sellSignal)}
	if data == nil {
		return meta
	}
	meta.PrefixBarsBeforeWindow = preAnalysisWindowBarCount(data, analysisMonths)
	ClampProbeSeriesToMonthsBack(data, analysisMonths)
	if meta.RequiredWarmupBars <= 0 {
		attachIndicatorWarmupFields(data, meta)
		return meta
	}
	if meta.PrefixBarsBeforeWindow < meta.RequiredWarmupBars {
		deficit := meta.RequiredWarmupBars - meta.PrefixBarsBeforeWindow
		meta.SuppressedInWindowBars = suppressLeadingProbeSignals(data, deficit)
		meta.Note = fmt.Sprintf(
			"指标预热不足（窗口前仅 %d 根 K 线，需要 %d）：已抑制分析窗内前 %d 根 K 线的买卖信号，避免 MACD/SAR 等未收敛误触发。",
			meta.PrefixBarsBeforeWindow, meta.RequiredWarmupBars, meta.SuppressedInWindowBars,
		)
	}
	attachIndicatorWarmupFields(data, meta)
	return meta
}

func attachIndicatorWarmupFields(data map[string]any, meta IndicatorWarmupMeta) {
	if data == nil {
		return
	}
	if meta.RequiredWarmupBars > 0 {
		data["indicator_warmup_bars"] = meta.RequiredWarmupBars
	}
	data["indicator_warmup_prefix_bars"] = meta.PrefixBarsBeforeWindow
	if meta.SuppressedInWindowBars > 0 {
		data["indicator_warmup_suppressed_bars"] = meta.SuppressedInWindowBars
	}
	if meta.Note != "" {
		data["indicator_warmup_note"] = meta.Note
	}
}

func suppressLeadingProbeSignals(data map[string]any, n int) int {
	if data == nil || n <= 0 {
		return 0
	}
	bars, _ := data["bars"].([]any)
	if len(bars) == 0 {
		return 0
	}
	if n > len(bars) {
		n = len(bars)
	}
	suppressed := 0
	if buyM, ok := data["buy_merged"].([]any); ok && len(buyM) == len(bars) {
		for i := 0; i < n; i++ {
			if int(slots.FloatAny(buyM[i])) != 0 {
				suppressed++
			}
			buyM[i] = 0
		}
		data["buy_merged"] = buyM
	}
	if sellM, ok := data["sell_merged"].([]any); ok && len(sellM) == len(bars) {
		for i := 0; i < n; i++ {
			if int(slots.FloatAny(sellM[i])) != 0 {
				suppressed++
			}
			sellM[i] = 0
		}
		data["sell_merged"] = sellM
	}
	oldLen := len(bars)
	if rules, ok := data["buy_rules"].([]any); ok {
		data["buy_rules"] = zeroLeadingSignalRules(rules, n, oldLen)
	}
	if rules, ok := data["sell_rules"].([]any); ok {
		data["sell_rules"] = zeroLeadingSignalRules(rules, n, oldLen)
	}
	if buyM, ok := data["buy_merged"].([]any); ok {
		data["buy_hits"] = countMergedSignalHits(buyM, 1)
	}
	if sellM, ok := data["sell_merged"].([]any); ok {
		data["sell_hits"] = countMergedSignalHits(sellM, -1)
	}
	return n
}

func zeroLeadingSignalRules(rules []any, n, barLen int) []any {
	out := make([]any, 0, len(rules))
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		cp := map[string]any{}
		for k, v := range rule {
			cp[k] = v
		}
		if ser, ok := rule["signal_series"].([]any); ok && len(ser) == barLen && n > 0 {
			cpSer := append([]any(nil), ser...)
			for i := 0; i < n && i < len(cpSer); i++ {
				cpSer[i] = 0
			}
			cp["signal_series"] = cpSer
		}
		out = append(out, cp)
	}
	return out
}
