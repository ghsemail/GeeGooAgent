package priceanalysis

import "fmt"

func barOHLCV(i int, high, low, close, vol float64) map[string]any {
	return map[string]any{
		"time":   fmt.Sprintf("2026-01-%02d 10:00:00", (i%28)+1),
		"high":   high,
		"low":    low,
		"close":  close,
		"volume": vol,
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
