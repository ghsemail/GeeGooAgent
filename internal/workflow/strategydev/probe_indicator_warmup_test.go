package strategydev

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

var sarMacdBuy = []any{
	map[string]any{"index": "SAR", "type": "flag", "param": map[string]any{}},
	map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
}

func TestIndicatorWarmupBarCountSARMACD(t *testing.T) {
	got := indicatorWarmupBarCount(sarMacdBuy, nil)
	if got != minIndicatorWarmupBars {
		t.Fatalf("warmup=%d want %d", got, minIndicatorWarmupBars)
	}
}

func TestEnrichProbeArgsExtends60mFetch(t *testing.T) {
	args := map[string]any{
		"code": "TSLA.US", "frequency": "60m", "months_back": 3,
	}
	w := EnrichProbeArgsForIndicatorWarmup(args, sarMacdBuy, nil)
	if w != minIndicatorWarmupBars {
		t.Fatalf("warmup=%d", w)
	}
	months := intFromAnyArg(args["months_back"])
	if months <= 3 {
		t.Fatalf("months_back=%d want >3 for warmup extension", months)
	}
	limit := intFromAnyArg(args["limit"])
	if limit < estimatedAnalysisBarCount(3, "60m")+minIndicatorWarmupBars {
		t.Fatalf("limit=%d too small", limit)
	}
}

func TestJul17FalseBuyWithoutWarmupPrefixSuppressed(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 17, 11, 30, 0, 0, loc)
	windowStart := windowEnd.AddDate(0, -3, 0)

	var bars []any
	var buyM []any
	day := windowStart
	closeP := 380.0
	for !day.After(windowEnd) {
		for _, hm := range []struct{ h, m int }{{10, 30}, {11, 30}, {14, 0}, {15, 0}} {
			ts := time.Date(day.Year(), day.Month(), day.Day(), hm.h, hm.m, 0, 0, loc)
			if ts.Before(windowStart) || ts.After(windowEnd) {
				continue
			}
			bars = append(bars, map[string]any{
				"time":   ts.Format("200601021504") + " 00:00:00",
				"close":  closeP,
				"high":   closeP + 1,
				"low":    closeP - 1,
				"volume": 1000.0,
			})
			buyM = append(buyM, 0)
			closeP += 0.01
		}
		day = day.AddDate(0, 0, 1)
	}
	// Unconverged MACD hist=0 buy on 2026-07-17 11:30 (3rd in-window bar); no warmup prefix in payload.
	if len(bars) < 3 {
		t.Fatalf("bars=%d", len(bars))
	}
	buyM[2] = 1

	data := map[string]any{"bars": bars, "buy_merged": buyM, "sell_merged": make([]any, len(bars)), "buy_hits": 1}
	meta := FinalizeDiagnoseProbeSeries(data, 3, sarMacdBuy, nil)
	if meta.SuppressedInWindowBars == 0 {
		t.Fatalf("expected suppression with short prefix, meta=%+v", meta)
	}
	outBuy, _ := data["buy_merged"].([]any)
	for i, raw := range outBuy {
		if int(slots.FloatAny(raw)) == 1 {
			ts := barTimeUnix(data["bars"].([]any)[i].(map[string]any))
			t.Fatalf("buy still present at %v after warmup finalize", ts.In(loc))
		}
	}
	if intFromAnyArg(data["buy_hits"]) != 0 {
		t.Fatalf("buy_hits=%v want 0", data["buy_hits"])
	}
}

func TestJul17GenuineBuyKeptWithWarmPrefix(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 17, 11, 30, 0, 0, loc)
	windowStart := windowEnd.AddDate(0, -3, 0)

	var bars []any
	var buyM []any
	day := windowStart.AddDate(0, 0, -40)
	closeP := 370.0
	for !day.After(windowEnd) {
		for _, hm := range []struct{ h, m int }{{10, 30}, {11, 30}, {14, 0}, {15, 0}} {
			ts := time.Date(day.Year(), day.Month(), day.Day(), hm.h, hm.m, 0, 0, loc)
			if ts.After(windowEnd) {
				break
			}
			bars = append(bars, map[string]any{
				"time":   ts.Format("200601021504") + " 00:00:00",
				"close":  closeP,
				"high":   closeP + 1,
				"low":    closeP - 1,
				"volume": 1000.0,
			})
			buyM = append(buyM, 0)
			closeP += 0.01
		}
		day = day.AddDate(0, 0, 1)
	}
	genuine := time.Date(2026, 7, 21, 10, 30, 0, 0, loc)
	for i, raw := range bars {
		ts := barTimeUnix(raw.(map[string]any))
		if ts.Equal(genuine) {
			buyM[i] = 1 // converged hist +0.08 after warmup fetch on Signal
		}
	}

	data := map[string]any{"bars": bars, "buy_merged": buyM, "sell_merged": make([]any, len(bars)), "buy_hits": 1}
	meta := FinalizeDiagnoseProbeSeries(data, 3, sarMacdBuy, nil)
	if meta.PrefixBarsBeforeWindow < minIndicatorWarmupBars {
		t.Fatalf("prefix=%d want >= %d", meta.PrefixBarsBeforeWindow, minIndicatorWarmupBars)
	}
	if meta.SuppressedInWindowBars != 0 {
		t.Fatalf("unexpected suppression meta=%+v", meta)
	}
	outBars, _ := data["bars"].([]any)
	outBuy, _ := data["buy_merged"].([]any)
	kept := false
	for i, raw := range outBuy {
		if int(slots.FloatAny(raw)) != 1 {
			continue
		}
		ts := barTimeUnix(outBars[i].(map[string]any))
		if ts.Equal(genuine) {
			kept = true
		}
	}
	if !kept {
		t.Fatal("genuine Jul 21 buy should remain after adequate warmup prefix")
	}
}

func TestClampOnlyWouldKeepJul17FalseBuy(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 10, 17, 11, 30, 0, 0, loc)
	windowStart := windowEnd.AddDate(0, -3, 0)
	var bars []any
	var buyM []any
	day := windowStart
	for !day.After(windowEnd) {
		for _, hm := range []struct{ h, m int }{{10, 30}, {11, 30}, {14, 0}, {15, 0}} {
			ts := time.Date(day.Year(), day.Month(), day.Day(), hm.h, hm.m, 0, 0, loc)
			if ts.Before(windowStart) || ts.After(windowEnd) {
				continue
			}
			bars = append(bars, map[string]any{
				"time": ts.Format("200601021504") + " 00:00:00", "close": 382.55,
				"high": 383.0, "low": 382.0, "volume": 1.0,
			})
			buyM = append(buyM, 0)
		}
		day = day.AddDate(0, 0, 1)
	}
	buyM[2] = 1
	data := map[string]any{"bars": bars, "buy_merged": buyM}
	ClampProbeSeriesToMonthsBack(data, 3)
	outBuy, _ := data["buy_merged"].([]any)
	if int(slots.FloatAny(outBuy[2])) != 1 {
		t.Fatal("baseline clamp-only should keep early false buy on 3rd in-window bar")
	}
	_ = loc
}
