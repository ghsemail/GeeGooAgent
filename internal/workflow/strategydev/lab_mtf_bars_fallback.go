package strategydev

import (
	"context"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
)

// mtfFallbackBarsFetcher uses probe MTF OHLCV when lab-api FetchBars fails or returns empty.
// K-line MTF and PA share the same upward TFs; probe path already succeeded for display.
type mtfFallbackBarsFetcher struct {
	Inner priceanalysis.BarsFetcher
	MTF   *KlinePatternMTFBundle
}

func wrapMTFFallbackBarsFetcher(inner priceanalysis.BarsFetcher, mtf *KlinePatternMTFBundle) priceanalysis.BarsFetcher {
	if inner == nil {
		return nil
	}
	if mtf == nil || len(mtf.ByFreq) == 0 {
		return inner
	}
	return mtfFallbackBarsFetcher{Inner: inner, MTF: mtf}
}

func (f mtfFallbackBarsFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	bars, ok := f.Inner.FetchBars(ctx, code, freq, months, asOf)
	if ok && len(bars) > 0 {
		return bars, true
	}
	if fb, fbOK := mtfBarsForPriceAnalysis(f.MTF, freq); fbOK {
		return fb, true
	}
	return bars, ok
}

func mtfBarsForPriceAnalysis(mtf *KlinePatternMTFBundle, freq string) ([]any, bool) {
	if mtf == nil {
		return nil, false
	}
	norm := priceanalysis.NormalizeFrequency(freq)
	for key, ser := range mtf.ByFreq {
		if priceanalysis.NormalizeFrequency(key) != norm && priceanalysis.NormalizeFrequency(ser.Freq) != norm {
			continue
		}
		if ser.FetchOK && len(ser.Bars) > 0 {
			out := make([]any, len(ser.Bars))
			copy(out, ser.Bars)
			return out, true
		}
	}
	return nil, false
}

func dailyBarsForLabAnchorFromMTF(mtf *KlinePatternMTFBundle) []any {
	bars, ok := mtfBarsForPriceAnalysis(mtf, "daily")
	if !ok {
		return nil
	}
	return bars
}

var _ priceanalysis.BarsFetcher = mtfFallbackBarsFetcher{}
