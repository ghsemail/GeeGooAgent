package strategydev

import (
	"testing"
)

func barOHLCV(i int, high, low, close, vol float64) map[string]any {
	return map[string]any{
		"high": high, "low": low, "close": close, "volume": vol,
	}
}

func TestPriceFilterInsufficientBarsUnclear(t *testing.T) {
	var bars []any
	for i := 0; i < 10; i++ {
		bars = append(bars, barOHLCV(i, 101, 99, 100, 1000))
	}
	pf := AnalyzePriceFilter(bars, "60m")
	if pf.Regime != RegimeUnclear {
		t.Fatalf("regime=%s want 不明确", pf.Regime)
	}
	if pf.AllowLong || pf.AllowShort {
		t.Fatal("should not allow directions")
	}
}

func TestPriceFilterCHOCMarksUnclear(t *testing.T) {
	bars := buildTrendBars(90, "up")
	// Force last bar to break below prior swing low (CHOCH in uptrend).
	last, _ := bars[len(bars)-1].(map[string]any)
	last["close"] = 80.0
	last["low"] = 79.0
	last["high"] = 81.0
	last["volume"] = 5000.0
	pf := AnalyzePriceFilter(bars, "60m")
	if pf.Regime != RegimeUnclear {
		t.Fatalf("regime=%s want 不明确 after CHOCH", pf.Regime)
	}
}

func TestPriceFilterLowVolumeBOSDowngrade(t *testing.T) {
	bars := buildTrendBars(90, "up")
	last, _ := bars[len(bars)-1].(map[string]any)
	last["close"] = 200.0
	last["high"] = 201.0
	last["low"] = 199.0
	last["volume"] = 1.0 // RVOL << 1 vs prior bars at 1000
	pf := AnalyzePriceFilter(bars, "60m")
	if pf.Regime != RegimeUnclear {
		t.Fatalf("regime=%s want 不明确 on suspicious low-volume BOS", pf.Regime)
	}
}

func TestMaskEpisodeTriggersStripsDisallowed(t *testing.T) {
	pf := PriceFilterResult{Regime: RegimeUnclear, AllowLong: false, AllowShort: false}
	buy := []any{0, 1, 0, 1}
	sell := []any{0, 0, -1, 0}
	nb, ns, st := MaskEpisodeTriggers(buy, sell, pf)
	if st.StrippedBuy != 2 || st.StrippedSell != 1 {
		t.Fatalf("stripped buy=%d sell=%d", st.StrippedBuy, st.StrippedSell)
	}
	for _, v := range nb {
		if int(v.(int)) == 1 {
			t.Fatal("buy should be cleared")
		}
	}
	for _, v := range ns {
		if int(v.(int)) == -1 {
			t.Fatal("sell should be cleared")
		}
	}
}

func TestUsePriceAnalysisEpisodeFilterDefaultOn(t *testing.T) {
	if !UsePriceAnalysisEpisodeFilterEnabled(nil) {
		t.Fatal("nil should default to enabled")
	}
	off := false
	if UsePriceAnalysisEpisodeFilterEnabled(&off) {
		t.Fatal("explicit false should disable")
	}
}

func buildTrendBars(n int, dir string) []any {
	bars := make([]any, 0, n)
	base := 100.0
	for i := 0; i < n; i++ {
		wave := float64((i / 3) % 4)
		if dir == "down" {
			wave = -wave
		}
		c := base + wave*2 + float64(i)*0.05
		bars = append(bars, barOHLCV(i, c+1.5, c-1.5, c, 1000))
	}
	return bars
}
