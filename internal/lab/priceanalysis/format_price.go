package priceanalysis

import (
	"math"
	"strconv"
	"strings"
)

// FormatLabPrice formats OHLC/swing prices for lab PA display (never scientific notation).
func FormatLabPrice(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	if v == 0 {
		return "0.00"
	}
	abs := math.Abs(v)
	prec := 2
	switch {
	case abs < 0.01:
		prec = 6
	case abs < 1:
		prec = 4
	case abs >= 10000:
		prec = 2
	}
	s := strconv.FormatFloat(v, 'f', prec, 64)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

// FormatLabSpotLabel returns the standard spot price caption.
func FormatLabSpotLabel(price float64) string {
	return "现价 " + FormatLabPrice(price)
}
