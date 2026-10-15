package strategydev

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestClampProbeSeriesToMonthsBackCNCompactTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 15, 11, 30, 0, 0, loc)
	windowStart := windowEnd.AddDate(0, -3, 0)

	var bars []any
	var buyM []any
	day := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	for i := 0; !day.After(windowEnd); i++ {
		compact := day.Format("200601021504") + " 00:00:00"
		bars = append(bars, map[string]any{
			"time": compact, "close": 100.0 + float64(i), "high": 101.0, "low": 99.0, "volume": 1.0,
		})
		hit := 0
		if i == 0 {
			hit = 1
		}
		if day.Equal(windowStart.Add(24 * time.Hour)) {
			hit = 1
		}
		buyM = append(buyM, hit)
		day = day.AddDate(0, 0, 1)
	}
	sellM := make([]any, len(bars))
	signalSeries := make([]any, len(bars))
	valueSeries := make([]any, len(bars))
	for i := range bars {
		signalSeries[i] = float64(i)
		valueSeries[i] = float64(i) * 0.1
	}
	data := map[string]any{
		"bars": bars, "buy_merged": buyM, "sell_merged": sellM,
		"buy_rules": []any{map[string]any{
			"index": "MACD", "signal_series": signalSeries,
			"value_series": map[string]any{"macd": valueSeries},
		}},
	}
	ClampProbeSeriesToMonthsBack(data, 3)
	outBars, _ := data["bars"].([]any)
	if len(outBars) == 0 {
		t.Fatal("expected bars after clamp")
	}
	firstTs := barTimeUnix(outBars[0].(map[string]any))
	if firstTs.Before(windowStart) {
		t.Fatalf("first bar %v before window start %v", firstTs.In(loc), windowStart)
	}
	lastTs := barTimeUnix(outBars[len(outBars)-1].(map[string]any))
	if lastTs.After(windowEnd) {
		t.Fatalf("last bar %v after window end %v", lastTs.In(loc), windowEnd)
	}
	outBuy, _ := data["buy_merged"].([]any)
	if len(outBuy) != len(outBars) {
		t.Fatalf("buy_merged len=%d bars=%d", len(outBuy), len(outBars))
	}
	for i, raw := range outBuy {
		if int(raw.(int)) == 1 {
			ts := barTimeUnix(outBars[i].(map[string]any))
			if ts.Before(windowStart) {
				t.Fatalf("buy trigger at %v before window", ts.In(loc))
			}
		}
	}
	lb, _ := data["bars_lookback"].([]any)
	if len(lb) <= len(outBars) {
		t.Fatalf("lookback len=%d want > clamped %d", len(lb), len(outBars))
	}
}

func TestOhlcvSourceForTimeframePrefersLongerPreAnchorHistory(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 7, 17, 16, 0, 0, 0, loc)
	short := []any{
		map[string]any{"time": "202607151500 00:00:00", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0},
		map[string]any{"time": "202607161130 00:00:00", "close": 2.0, "high": 2.1, "low": 1.9, "volume": 1.0},
		map[string]any{"time": "202607171130 00:00:00", "close": 3.0, "high": 3.1, "low": 2.9, "volume": 1.0},
	}
	var long []any
	day := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	for !day.After(anchor) {
		long = append(long, map[string]any{
			"time": day.Format("200601021504") + " 00:00:00",
			"close": 10.0, "high": 11.0, "low": 9.0, "volume": 1.0,
		})
		day = day.AddDate(0, 0, 1)
	}
	src := ohlcvSourceForTimeframe("60m", "60m", map[string][]any{"60m": short}, long, anchor)
	if len(truncateBarsAtAnchor(src, anchor)) < minBarsForPriceFilter("60m") {
		t.Fatalf("src pre-anchor bars=%d want >= %d", len(truncateBarsAtAnchor(src, anchor)), minBarsForPriceFilter("60m"))
	}
}

func append60mSessionBars(out []any, day time.Time, baseClose float64) []any {
	loc := day.Location()
	sessions := []struct{ h, m int }{{10, 30}, {11, 30}, {14, 0}, {15, 0}}
	for _, s := range sessions {
		ts := time.Date(day.Year(), day.Month(), day.Day(), s.h, s.m, 0, 0, loc)
		out = append(out, map[string]any{
			"time": ts.Format("200601021504") + " 00:00:00",
			"close": baseClose, "high": baseClose + 1, "low": baseClose - 1, "volume": 1000.0,
		})
		baseClose += 0.01
	}
	return out
}

func TestPerSignal60mUsesLookbackWhenFirstTriggerNearChartStart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 15, 15, 0, 0, 0, loc)
	var lookback []any
	day := time.Date(2026, 4, 29, 0, 0, 0, 0, loc)
	closeP := 400.0
	for !day.After(windowEnd) {
		lookback = append60mSessionBars(lookback, day, closeP)
		closeP += 0.2
		day = day.AddDate(0, 0, 1)
	}
	data := map[string]any{"bars": lookback}
	ClampProbeSeriesToMonthsBack(data, 3)
	clamped, _ := data["bars"].([]any)
	buyM := make([]any, len(clamped))
	buyM[5] = 1 // a few bars after clamped chart start (~07-20)

	short60m := make([]any, len(clamped))
	for i, raw := range clamped {
		short60m[i] = raw
	}
	dailyFull := make([]any, minBarsForPriceFilter("daily")+15)
	for j := range dailyFull {
		d := time.Date(2026, 5, 1, 0, 0, 0, 0, loc).AddDate(0, 0, j)
		c := 400.0 + float64(j)*0.2
		dailyFull[j] = map[string]any{
			"time": d.Format("2006-01-02 15:04:05"),
			"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}

	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		switch freq {
		case "daily":
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": dailyFull}}
		case "60m":
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": short60m}}
		default:
			n := minBarsForPriceFilter(freq) + 5
			out := make([]any, n)
			for i := 0; i < n; i++ {
				c := 100.0 + float64(i)*0.05
				out[i] = map[string]any{
					"time": fmt.Sprintf("2026-06-%02d 10:00:00", (i%28)+1),
					"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
				}
			}
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": out}}
		}
	})

	row := barRowAt(clamped, 5)
	anchor, _, ok := anchorForSignalBar(row, dailyFull)
	if !ok {
		t.Fatal("anchor missing in test fixture")
	}
	if n := len(truncateBarsAtAnchor(lookback, anchor)); n < minBarsForPriceFilter("60m") {
		t.Fatalf("fixture lookback trunc=%d want >= %d", n, minBarsForPriceFilter("60m"))
	}
	chosen := ohlcvSourceForTimeframe("60m", "60m", map[string][]any{"60m": short60m}, lookback, anchor)
	if len(truncateBarsAtAnchor(chosen, anchor)) < minBarsForPriceFilter("60m") {
		t.Fatalf("chosen src trunc=%d", len(truncateBarsAtAnchor(chosen, anchor)))
	}

	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "300750.SZ", Months: 3, AnnotateOnly: true, LookbackChartBars: lookback,
	}, clamped, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	tr := stats.BuyTriggers[0]
	if tr.Intercepted {
		t.Fatal("soft gate must not set intercepted")
	}
	if stats.StrippedBuy != 0 {
		t.Fatalf("stripped_buy=%d want 0 in soft mode", stats.StrippedBuy)
	}
	pa := tr.PriceAnalysis
	if pa == nil {
		t.Fatal("missing price analysis")
	}
	for _, row := range pa.Rows {
		if row.Timeframe != "60m" {
			continue
		}
		if row.Structure == structureInsufficientBars {
			t.Fatalf("60m structure=%q with lookback span", row.Structure)
		}
	}
}

func TestPerSignalFetchNotCappedAtSixWhenChartNeedsLookback(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 15, 11, 30, 0, 0, loc)
	var bars []any
	var buyM []any
	day := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	for i := 0; !day.After(windowEnd); i++ {
		bars = append(bars, map[string]any{
			"time": day.Format("200601021504") + " 00:00:00",
			"close": 400.0 + float64(i)*0.1, "high": 401.0, "low": 399.0, "volume": 1000.0,
		})
		buyM = append(buyM, 0)
		day = day.AddDate(0, 0, 1)
	}
	buyM[0] = 1

	gotDaily := monthsBackForPerSignalPriceFetch(3, bars, "daily", windowEnd)
	if gotDaily <= 6 {
		t.Fatalf("monthsBackForPerSignalPriceFetch daily=%d want >6 for Apr–Oct chart + anchor lookback", gotDaily)
	}
	args := PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{
		Code: "300750.SZ", Freq: "daily", Months: gotDaily, ExtendedFetch: true,
	})
	if args["months_back"].(int) <= 6 {
		t.Fatalf("ExtendedFetch probe months_back=%v want >6", args["months_back"])
	}

	dailyFull := make([]any, minBarsForPriceFilter("daily")+10)
	for j := range dailyFull {
		day := time.Date(2026, 2, 1, 0, 0, 0, 0, loc).AddDate(0, 0, j)
		c := 400.0 + float64(j)*0.2
		dailyFull[j] = map[string]any{
			"time": day.Format("2006-01-02 15:04:05"),
			"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}
	mkBars := func(freq string, minLen int) []any {
		out := make([]any, minLen)
		for i := 0; i < minLen; i++ {
			c := 100.0 + float64(i)*0.05
			out[i] = map[string]any{
				"time": fmt.Sprintf("2026-05-%02d 10:00:00", (i%28)+1),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		return out
	}
	monthsByFreq := map[string]int{}
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		monthsByFreq[freq] = args["months_back"].(int)
		var b []any
		switch freq {
		case "daily":
			if args["months_back"].(int) <= 6 {
				b = []any{map[string]any{"time": "2026-09-01", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0}}
			} else {
				b = dailyFull
			}
		default:
			b = mkBars(freq, minBarsForPriceFilter(freq)+5)
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": b}}
	})

	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "300750.SZ", Months: 3, LookbackChartBars: bars,
	}, bars, "60m", buyM, nil)
	if monthsByFreq["daily"] <= 6 {
		t.Fatalf("daily months_back=%d want >6 (old CapMonthsBack=6 on fetch starved lookback)", monthsByFreq["daily"])
	}
	if len(stats.BuyTriggers) < 1 {
		t.Fatalf("buy triggers=%d", len(stats.BuyTriggers))
	}
	for _, tr := range stats.BuyTriggers {
		pa := tr.PriceAnalysis
		if pa == nil {
			t.Fatalf("missing price analysis idx=%d", tr.Index)
		}
		for _, row := range pa.Rows {
			if row.Timeframe == "60m" || row.Timeframe == "日线" {
				if row.Structure == structureInsufficientBars {
					t.Fatalf("trigger idx=%d %s structure=%q", tr.Index, row.Timeframe, row.Structure)
				}
			}
		}
	}
}
