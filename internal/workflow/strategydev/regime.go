package strategydev

import (
	"fmt"
	"math"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

type regimeOutcome struct {
	Regime     string
	Side       string
	Confidence float64
	Note       string
}

const (
	adxTrendThreshold = 25.0
	adxRangeThreshold = 20.0
)

// ClassifyRegime applies locked ADX + DI rules on the last bar of aligned indicator series.
func ClassifyRegime(bars []any, series map[string][]float64, prev *regimeOutcome) regimeOutcome {
	if len(bars) < 2 {
		return regimeOutcome{Regime: RegimeUnclear, Side: SideNone, Note: "指标尚未预热"}
	}
	adx := pickSeries(series, "ADX", "ADX_ADX")
	plus := pickSeries(series, "PLUS_DI", "ADX_PLUS_DI", "P_DI", "+DI")
	minus := pickSeries(series, "MINUS_DI", "ADX_MINUS_DI", "M_DI", "-DI")
	upper, lower, mid := bollingerBands(series)
	if len(adx) == 0 || len(plus) == 0 || len(minus) == 0 {
		return regimeOutcome{Regime: RegimeUnclear, Side: SideNone, Note: "ADX/DI 未就绪"}
	}
	i := len(adx) - 1
	if i >= len(plus) || i >= len(minus) {
		i = minInt(len(adx), len(plus), len(minus)) - 1
	}
	if i < 0 {
		return regimeOutcome{Regime: RegimeUnclear, Side: SideNone, Note: "指标尚未预热"}
	}
	a := adx[i]
	pdi := plus[i]
	mdi := minus[i]
	if a <= 0 || pdi <= 0 && mdi <= 0 {
		return regimeOutcome{Regime: RegimeUnclear, Side: SideNone, Note: "指标尚未预热"}
	}

	var out regimeOutcome
	switch {
	case a >= adxTrendThreshold && pdi > mdi:
		out = regimeOutcome{Regime: RegimeUp, Side: SideRight, Confidence: 0.85}
	case a >= adxTrendThreshold && mdi > pdi:
		out = regimeOutcome{Regime: RegimeDown, Side: SideRight, Confidence: 0.85}
	case a < adxRangeThreshold:
		if bbRevertTouch(bars, upper, lower, mid, 3) {
			out = regimeOutcome{Regime: RegimeRevert, Side: SideLeft, Confidence: 0.7}
		} else {
			out = regimeOutcome{Regime: RegimeRange, Side: SideNone, Confidence: 0.65}
		}
	case a >= adxRangeThreshold && a < adxTrendThreshold:
		if prev != nil && prev.Regime != "" && prev.Regime != RegimeUnclear {
			out = regimeOutcome{Regime: prev.Regime, Side: prev.Side, Confidence: 0.45, Note: "ADX 灰色区间沿用上一根"}
		} else if pdi > mdi {
			out = regimeOutcome{Regime: RegimeWeakUp, Side: SideRight, Confidence: 0.4}
		} else {
			out = regimeOutcome{Regime: RegimeWeakDown, Side: SideRight, Confidence: 0.4}
		}
	default:
		out = regimeOutcome{Regime: RegimeUnclear, Side: SideNone}
	}

	if swingConflictsDI(bars, pdi, mdi, a) {
		out.Regime = RegimeUnclear
		out.Side = SideNone
		out.Note = "摆动结构与 DI 方向相反"
		out.Confidence = 0.3
	}
	out.Confidence = adjustConfidence(out, series, bars)
	return out
}

func adjustConfidence(base regimeOutcome, series map[string][]float64, bars []any) float64 {
	conf := base.Confidence
	if ema := pickSeries(series, "EMA", "EMA_LONG"); len(ema) > 1 {
		if ema[len(ema)-1] > ema[len(ema)-2] && base.Regime == RegimeUp {
			conf += 0.05
		}
		if ema[len(ema)-1] < ema[len(ema)-2] && base.Regime == RegimeDown {
			conf += 0.05
		}
	}
	if conf > 0.95 {
		return 0.95
	}
	if conf < 0.1 {
		return 0.1
	}
	return conf
}

func pickSeries(series map[string][]float64, keys ...string) []float64 {
	for _, k := range keys {
		if v, ok := series[strings.ToUpper(k)]; ok && len(v) > 0 {
			return v
		}
	}
	for key, vals := range series {
		uk := strings.ToUpper(key)
		for _, want := range keys {
			if strings.Contains(uk, strings.ToUpper(want)) {
				return vals
			}
		}
	}
	return nil
}

func bollingerBands(series map[string][]float64) (upper, lower, mid []float64) {
	upper = pickSeries(series, "UPPER", "BBAND_UPPER", "BB_UPPER")
	lower = pickSeries(series, "LOWER", "BBAND_LOWER", "BB_LOWER")
	mid = pickSeries(series, "MID", "MIDDLE", "BBAND_MIDDLE", "BB_MIDDLE")
	return upper, lower, mid
}

func bbRevertTouch(bars []any, upper, lower, mid []float64, lookback int) bool {
	if len(bars) == 0 || len(upper) == 0 || len(lower) == 0 {
		return false
	}
	n := len(bars)
	start := n - lookback
	if start < 0 {
		start = 0
	}
	touched := false
	for i := start; i < n; i++ {
		row, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		h := slots.FloatAny(row["high"])
		l := slots.FloatAny(row["low"])
		c := slots.FloatAny(row["close"])
		ui := seriesAt(upper, i)
		li := seriesAt(lower, i)
		mi := seriesAt(mid, i)
		if ui > 0 && (h >= ui || c >= ui) {
			touched = true
		}
		if li > 0 && (l <= li || c <= li) {
			touched = true
		}
		if !touched && mi > 0 && math.Abs(c-mi)/mi < 0.002 {
			touched = true
		}
	}
	if !touched {
		return false
	}
	last, ok := bars[n-1].(map[string]any)
	if !ok {
		return false
	}
	c := slots.FloatAny(last["close"])
	ui := seriesAt(upper, n-1)
	li := seriesAt(lower, n-1)
	mi := seriesAt(mid, n-1)
	if mi > 0 && math.Abs(c-mi)/mi < 0.01 {
		return true
	}
	if ui > 0 && li > 0 && c > li && c < ui {
		return true
	}
	return false
}

func seriesAt(vals []float64, i int) float64 {
	if len(vals) == 0 {
		return 0
	}
	if i >= len(vals) {
		i = len(vals) - 1
	}
	if i < 0 {
		return 0
	}
	return vals[i]
}

func swingConflictsDI(bars []any, pdi, mdi, adx float64) bool {
	if adx < adxTrendThreshold {
		return false
	}
	highs, lows := swingHighsLows(bars, 5)
	if len(highs) < 2 || len(lows) < 2 {
		return false
	}
	upHighs := highs[len(highs)-1] > highs[len(highs)-2]
	upLows := lows[len(lows)-1] > lows[len(lows)-2]
	bullStruct := upHighs && upLows
	bearStruct := !upHighs && !upLows
	if bullStruct && mdi > pdi {
		return true
	}
	if bearStruct && pdi > mdi {
		return true
	}
	return false
}

func swingHighsLows(bars []any, lookback int) (highs, lows []float64) {
	if lookback <= 0 {
		lookback = 3
	}
	start := len(bars) - lookback
	if start < 0 {
		start = 0
	}
	for i := start; i < len(bars); i++ {
		row, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		highs = append(highs, slots.FloatAny(row["high"]))
		lows = append(lows, slots.FloatAny(row["low"]))
	}
	return highs, lows
}

func minInt(vals ...int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func sideForRegime(regime string) string {
	switch regime {
	case RegimeUp, RegimeDown, RegimeWeakUp, RegimeWeakDown:
		return SideRight
	case RegimeRevert:
		return SideLeft
	default:
		return SideNone
	}
}

func regimeAllowsSelection(regime string) bool {
	return regime != RegimeRange && regime != RegimeUnclear
}

func formatRegimeNote(o regimeOutcome) string {
	if o.Note != "" {
		return o.Note
	}
	return fmt.Sprintf("%s · %s · 置信度 %.0f%%", o.Regime, o.Side, o.Confidence*100)
}
