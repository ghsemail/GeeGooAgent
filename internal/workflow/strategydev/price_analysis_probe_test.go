package strategydev

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPriceAnalysisProbeArgsOmitsAsOfWhenUnset(t *testing.T) {
	args := PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{Code: "00700.HK", Freq: "5m", Months: 3})
	if _, ok := args["as_of"]; ok {
		t.Fatalf("as_of must be omitted when unset: %v", args)
	}
}

func TestPriceAnalysisProbeArgsIncludesAsOfWhenSet(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	asOf := time.Date(2026, 10, 13, 16, 0, 0, 0, loc)
	args := PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{
		Code: "00700.HK", Freq: "5m", Months: 3, AsOf: &asOf,
	})
	if args["as_of"] != "2026-10-13 16:00:00" {
		t.Fatalf("as_of=%v", args["as_of"])
	}
}

func TestEpisodeFilterOneProbePerFrequencyWithAsOf(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 40
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		bars[i] = map[string]any{
			"time": time.Date(2026, 9, 1, 10, 0, 0, 0, loc).AddDate(0, 0, i).Format("2006-01-02 15:04:05"),
			"close": 400.0, "high": 401, "low": 399, "volume": 1000.0,
		}
	}
	buyM[10] = 1
	paProbeCount := 0
	emaProbeCount := 0
	asOfSeen := map[string]string{}
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		if isEMAStackProbeArgs(args) {
			emaProbeCount++
		} else {
			paProbeCount++
		}
		freq := args["frequency"].(string)
		if v, ok := args["as_of"].(string); ok {
			asOfSeen[freq] = v
		}
		minLen := minBarsForPriceFilter(freq) + 5
		out := make([]any, minLen)
		for i := range out {
			c := 100.0 + float64(i)
			out[i] = map[string]any{
				"time": fmtDaily(i), "close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": out}}
	})
	ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "00700.HK", Months: 3,
	}, bars, "60m", buyM, nil)
	if paProbeCount != 3 {
		t.Fatalf("PA probe calls=%d want 3 (60m, daily, weekly)", paProbeCount)
	}
	if emaProbeCount != 3 {
		t.Fatalf("EMA probe calls=%d want 3 (weekly/daily/60m EMA stack)", emaProbeCount)
	}
	if len(asOfSeen) != 3 {
		t.Fatalf("expected as_of on all probes, got %v", asOfSeen)
	}
	if _, ok := asOfSeen["5m"]; ok {
		t.Fatal("60m gate must not probe 5m")
	}
}

func TestEpisodeFilter00700JulyAnchor5mNotEmptyBeforeAnchor(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// 60m diagnose chart spanning into Oct; signal on 2026-07-27 references anchor 2026-07-24.
	n := 80
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		ts := time.Date(2026, 7, 1, 10, 0, 0, 0, loc).AddDate(0, 0, i)
		if i == n-1 {
			ts = time.Date(2026, 10, 10, 11, 0, 0, 0, loc)
		}
		bars[i] = map[string]any{
			"time": ts.Format("2006-01-02 15:04:05"),
			"close": 430.0 + float64(i)*0.1, "high": 431, "low": 429, "volume": 1e5,
		}
	}
	bars[5] = map[string]any{"time": "2026-07-27 10:30:00", "close": 430.0, "high": 431, "low": 429, "volume": 1e5}
	buyM[5] = 1
	daily := []any{
		map[string]any{"time": "2026-07-23 00:00:00", "close": 420.0, "high": 421, "low": 419, "volume": 1e6},
		map[string]any{"time": "2026-07-24 00:00:00", "close": 428.0, "high": 429, "low": 427, "volume": 1e6},
		map[string]any{"time": "2026-10-13 00:00:00", "close": 440.0, "high": 441, "low": 439, "volume": 1e6},
	}
	mk5mThroughJuly := func() []any {
		var out []any
		for d := 1; d <= 24; d++ {
			for h := 10; h <= 15; h++ {
				for m := 0; m < 60; m += 5 {
					if h == 15 && m > 0 {
						continue
					}
					ts := time.Date(2026, 7, d, h, m, 0, 0, loc)
					out = append(out, map[string]any{
						"time": ts.Format("2006-01-02 15:04:05"),
						"close": 400.0 + float64(len(out))*0.01, "high": 401, "low": 399, "volume": 1000.0,
					})
				}
			}
		}
		for i := 0; i < minBarsForPriceFilter("5m")+20; i++ {
			ts := time.Date(2026, 10, 1, 10, 0, 0, 0, loc).Add(time.Duration(i*5) * time.Minute)
			out = append(out, map[string]any{
				"time": ts.Format("2006-01-02 15:04:05"),
				"close": 435.0, "high": 436, "low": 434, "volume": 1000.0,
			})
		}
		return out
	}
	full5m := mk5mThroughJuly()
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		freq := args["frequency"].(string)
		var b []any
		switch freq {
		case "5m":
			b = full5m
		case "daily":
			b = daily
		default:
			b = full5m
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": b}}
	})
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "00700.HK", Months: 3,
	}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	pa := stats.BuyTriggers[0].PriceAnalysis
	if pa == nil {
		t.Fatal("missing pa")
	}
	wantAnchor := time.Date(2026, 7, 24, 16, 0, 0, 0, loc)
	if pa.AnchorClose != 428.0 {
		t.Fatalf("anchor close=%v want 428", pa.AnchorClose)
	}
	_ = wantAnchor
	for _, row := range pa.Rows {
		if row.Timeframe == "5m" {
			t.Fatalf("60m diagnose must not include 5m price_analysis row: %+v", row)
		}
	}
}

func TestBuildRunPriceAnalysisAsOfUsesProbeAsOf(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	asOf := time.Date(2026, 10, 13, 16, 0, 0, 0, loc)
	var sawAsOf bool
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if args["as_of"] == "2026-10-13 16:00:00" {
			sawAsOf = true
		}
		freq := args["frequency"].(string)
		n := minBarsForPriceFilter(freq) + 3
		out := make([]any, n)
		for i := range out {
			c := 100.0 + float64(i)
			out[i] = map[string]any{
				"time": time.Date(2026, 9, 1, 10, 0, 0, 0, loc).AddDate(0, 0, i).Format("2006-01-02"),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": out}}
	})
	pa := BuildRunPriceAnalysisAsOf(context.Background(), run, "00700.HK", asOf, 3, 0, KeyLevelBands{}, "60m")
	if !sawAsOf {
		t.Fatal("expected as_of on single-point probes")
	}
	if len(pa.Rows) != 3 {
		t.Fatalf("rows=%d want 3 for 60m filter", len(pa.Rows))
	}
	if pa.AnchorTime == "" {
		t.Fatal("expected anchor_time set for as_of run")
	}
}
