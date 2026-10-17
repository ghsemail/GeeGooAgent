package strategydev

import (
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const (
	emaStackLabelBull  = "多头"
	emaStackLabelBear  = "空头"
	emaStackLabelChop  = "纠缠"
	emaStackLabelUnknown = "unknown"
	emaPeriodFast      = 25
	emaPeriodMid       = 50
	emaPeriodSlow      = 120
	emaMinBarsForSlow  = 120
)

// EMAStackEntry is one timeframe's EMA25/50/120 stack at the price-analysis anchor.
type EMAStackEntry struct {
	Label string  `json:"label"`
	EMA25 float64 `json:"ema25,omitempty"`
	EMA50 float64 `json:"ema50,omitempty"`
	EMA120 float64 `json:"ema120,omitempty"`
	Close float64 `json:"close,omitempty"`
}

// EMAStacks holds weekly / daily / 60m stack labels for diagnose trigger records.
type EMAStacks struct {
	Weekly EMAStackEntry `json:"weekly"`
	Daily  EMAStackEntry `json:"daily"`
	M60    EMAStackEntry `json:"60m"`
}

var emaStackTimeframes = []struct {
	key  string
	freq string
}{
	{"weekly", "weekly"},
	{"daily", "daily"},
	{"60m", "60m"},
}

func unknownEMAStacks() EMAStacks {
	u := EMAStackEntry{Label: emaStackLabelUnknown}
	return EMAStacks{Weekly: u, Daily: u, M60: u}
}

// ComputeEMAStacksAtAnchor evaluates EMA stacks on bars truncated at anchorTime (display-only).
func ComputeEMAStacksAtAnchor(
	barsByFreq map[string][]any,
	fetchOKByFreq map[string]bool,
	lookbackBars []any,
	chartFreq string,
	anchorTime time.Time,
) EMAStacks {
	if anchorTime.IsZero() {
		return unknownEMAStacks()
	}
	out := EMAStacks{}
	for _, spec := range emaStackTimeframes {
		entry := computeEMAStackForTimeframe(spec.freq, spec.key, barsByFreq, fetchOKByFreq, lookbackBars, chartFreq, anchorTime)
		switch spec.key {
		case "weekly":
			out.Weekly = entry
		case "daily":
			out.Daily = entry
		case "60m":
			out.M60 = entry
		}
	}
	return out
}

func computeEMAStackForTimeframe(
	specFreq, _ string,
	barsByFreq map[string][]any,
	fetchOKByFreq map[string]bool,
	lookbackBars []any,
	chartFreq string,
	anchorTime time.Time,
) EMAStackEntry {
	freq := NormalizeFrequency(specFreq)
	if freq == "weekly" || freq == "60m" {
		if len(barsByFreq[freq]) == 0 && !fetchOKByFreq[freq] {
			return EMAStackEntry{Label: emaStackLabelUnknown}
		}
	}
	src := ohlcvSourceForTimeframe(freq, chartFreq, barsByFreq, lookbackBars, anchorTime)
	trunc := truncateBarsAtAnchor(src, anchorTime)
	if len(trunc) < emaMinBarsForSlow {
		return EMAStackEntry{Label: emaStackLabelUnknown}
	}
	closes := barClosesInOrder(trunc)
	if len(closes) < emaMinBarsForSlow {
		return EMAStackEntry{Label: emaStackLabelUnknown}
	}
	ef := lastRecursiveEMA(closes, emaPeriodFast)
	em := lastRecursiveEMA(closes, emaPeriodMid)
	es := lastRecursiveEMA(closes, emaPeriodSlow)
	close := closes[len(closes)-1]
	return EMAStackEntry{
		Label:  classifyEMAStackLabel(ef, em, es, close),
		EMA25:  roundEMAValue(ef),
		EMA50:  roundEMAValue(em),
		EMA120: roundEMAValue(es),
		Close:  roundEMAValue(close),
	}
}

func classifyEMAStackLabel(ef, em, es, close float64) string {
	if ef > em && em > es && close > em {
		return emaStackLabelBull
	}
	if ef < em && em < es && close < em {
		return emaStackLabelBear
	}
	return emaStackLabelChop
}

// recursiveEMASeries uses GeeGooSignal-style EMA: seed = first close, mult = 2/(period+1).
func recursiveEMASeries(closes []float64, period int) []float64 {
	if len(closes) == 0 || period <= 0 {
		return nil
	}
	mult := 2.0 / float64(period+1)
	out := make([]float64, len(closes))
	out[0] = closes[0]
	for i := 1; i < len(closes); i++ {
		out[i] = (closes[i]-out[i-1])*mult + out[i-1]
	}
	return out
}

func lastRecursiveEMA(closes []float64, period int) float64 {
	series := recursiveEMASeries(closes, period)
	if len(series) == 0 {
		return 0
	}
	return series[len(series)-1]
}

func roundEMAValue(v float64) float64 {
	return NormalizePriceScore(v)
}

func paSubscoreFromTFScores(tfScores map[string]TFScoreBreakdown, tfLabel string) *float64 {
	if tfScores == nil {
		return nil
	}
	br, ok := tfScores[tfLabel]
	if !ok || br.Missing {
		return nil
	}
	v := br.Subscore
	return &v
}

func barClosesInOrder(bars []any) []float64 {
	out := make([]float64, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c := slots.FloatAny(row["close"])
		if c == 0 && row["close"] == nil {
			continue
		}
		out = append(out, c)
	}
	return out
}
