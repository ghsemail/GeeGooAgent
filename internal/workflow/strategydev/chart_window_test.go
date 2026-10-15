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
			hit = 1 // April — outside 3mo window
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
	rules, _ := data["buy_rules"].([]any)
	rule := rules[0].(map[string]any)
	ss, _ := rule["signal_series"].([]any)
	if len(ss) != len(outBars) {
		t.Fatalf("signal_series len=%d want %d", len(ss), len(outBars))
	}
	vs, _ := rule["value_series"].(map[string]any)
	macd, _ := vs["macd"].([]any)
	if len(macd) != len(outBars) {
		t.Fatalf("value_series macd len=%d want %d", len(macd), len(outBars))
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
	buyM[0] = 1 // earliest trigger — reproduces 300750.SZ over-span before chart clamp

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
		Run: run, Code: "300750.SZ", Months: 3,
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
