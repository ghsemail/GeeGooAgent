package strategydev

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPerSignalFilterNoChartBarAnchorFallbackWhenDailyMissing(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 10
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		bars[i] = map[string]any{
			"time": time.Date(2026, 6, 29, 11, 0, 0, 0, loc).Format("2006-01-02 15:04:05"),
			"close": 428.2, "high": 429, "low": 427, "volume": 1000.0,
		}
	}
	buyM[5] = 1
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{Months: 3}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	pa := stats.BuyTriggers[0].PriceAnalysis
	if pa == nil {
		t.Fatal("missing price analysis")
	}
	if pa.AnchorClose != 0 || pa.AnchorTime != "" {
		t.Fatalf("must not use signal/chart anchor: close=%v time=%q", pa.AnchorClose, pa.AnchorTime)
	}
	if pa.SpotLabel == "信号收盘 428.2" || pa.AnchorClose == 428.2 {
		t.Fatalf("signal bar used as anchor: %+v", pa)
	}
}

func TestPerSignalFilterExtendsFetchMonthsForEarlyChartSignal(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 120
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		var ts string
		if i < n-1 {
			day := 24 + (i / 4)
			if day > 30 {
				day = 30
			}
			ts = fmt.Sprintf("2026-06-%02d 11:00:00", day)
		} else {
			ts = "2026-10-10 11:00:00"
		}
		bars[i] = map[string]any{
			"time": ts,
			"close": 400.0 + float64(i)*0.1, "high": 401.0, "low": 399.0, "volume": 1000.0,
		}
	}
	bars[7] = map[string]any{
		"time": "2026-06-24 11:00:00", "close": 428.8, "high": 430, "low": 427, "volume": 1000.0,
	}
	sellM[7] = -1
	bars[20] = map[string]any{
		"time": "2026-06-26 11:00:00", "close": 411.8, "high": 412, "low": 410, "volume": 1000.0,
	}
	buyM[20] = 1

	dailyFull := make([]any, minBarsForPriceFilter("daily")+5)
	for j := range dailyFull {
		day := j + 1
		c := 400.0 + float64(j)*0.3
		dailyFull[j] = map[string]any{
			"time": fmt.Sprintf("2026-04-%02d 00:00:00", (day%28)+1),
			"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}
	dailyFull[len(dailyFull)-4] = map[string]any{"time": "2026-06-23 00:00:00", "close": 420.0, "high": 421, "low": 419, "volume": 1e6}
	dailyFull[len(dailyFull)-3] = map[string]any{"time": "2026-06-24 00:00:00", "close": 428.8, "high": 430, "low": 427, "volume": 1e6}
	dailyFull[len(dailyFull)-2] = map[string]any{"time": "2026-06-25 00:00:00", "close": 415.0, "high": 416, "low": 414, "volume": 1e6}
	dailyFull[len(dailyFull)-1] = map[string]any{"time": "2026-06-26 00:00:00", "close": 411.8, "high": 412, "low": 410, "volume": 1e6}
	dailyShort := []any{
		map[string]any{"time": "2026-07-01 00:00:00", "close": 430.0, "high": 431, "low": 429, "volume": 1e6},
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
		var daily []any
		if months, ok := args["months_back"].(int); ok && months <= 3 {
			daily = dailyShort
		} else {
			daily = dailyFull
		}
		minLen := minBarsForPriceFilter(freq) + 5
		b := mkBars(freq, minLen)
		if freq == "daily" {
			b = daily
		}
		if freq == "weekly" {
			b = mkBars(freq, minLen)
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": b}}
	})

	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "00700.HK", Months: 3,
	}, bars, "60m", buyM, sellM)
	if len(stats.SellTriggers) != 1 || len(stats.BuyTriggers) != 1 {
		t.Fatalf("sell=%d buy=%d", len(stats.SellTriggers), len(stats.BuyTriggers))
	}
	if monthsByFreq["daily"] <= 3 {
		t.Fatalf("expected extended daily months_back, saw %v", monthsByFreq)
	}
	sellPA := stats.SellTriggers[0].PriceAnalysis
	if sellPA == nil || sellPA.AnchorClose != 420.0 {
		t.Fatalf("sell anchor close=%v want 420 (2026-06-23)", sellPA)
	}
	wantAnchor := time.Date(2026, 6, 23, 16, 0, 0, 0, loc)
	at, _, ok := anchorForSignalBar(barRowAt(bars, 7), dailyFull)
	if !ok || !at.Equal(wantAnchor) {
		t.Fatalf("anchor=%v ok=%v", at.In(loc), ok)
	}
	for _, tr := range []EpisodeTriggerRecord{stats.SellTriggers[0], stats.BuyTriggers[0]} {
		pa := tr.PriceAnalysis
		if pa == nil || len(pa.Rows) != 3 {
			t.Fatalf("rows=%d want 3 for 60m gate", len(pa.Rows))
		}
		for _, row := range pa.Rows {
			if row.Structure == structureNoBarsBeforeAnchor || row.Structure == structureInsufficientBars {
				t.Fatalf("%s structure=%q", row.Timeframe, row.Structure)
			}
		}
		if pa.AnchorClose == tr.Close {
			t.Fatalf("signal close used as anchor idx=%d", tr.Index)
		}
	}
}

func TestAnchorForSignalMondayUsesFridayDailyClose00700(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// Daily bars newest-first (common in API payloads) with 00:00:00 wall-clock strings.
	daily := []any{
		map[string]any{"time": "2026-06-29 00:00:00", "close": 428.2, "high": 430, "low": 425, "volume": 1e6},
		map[string]any{"time": "2026-06-26 00:00:00", "close": 411.8, "high": 415, "low": 410, "volume": 1e6},
	}
	signal := map[string]any{"time": "2026-06-29 12:00:00", "close": 428.2, "high": 429, "low": 427, "volume": 1e5}
	anchorTime, anchorClose, ok := anchorForSignalBar(signal, daily)
	if !ok {
		t.Fatal("expected prior trading day anchor")
	}
	if anchorClose != 411.8 {
		t.Fatalf("anchor close=%v want 411.8", anchorClose)
	}
	wantAnchor := time.Date(2026, 6, 26, 16, 0, 0, 0, loc)
	if !anchorTime.Equal(wantAnchor) {
		t.Fatalf("anchor instant=%v want %v", anchorTime.In(loc), wantAnchor)
	}
}

func TestAnchorForSignalMondayUsesFridayDaily(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	daily := []any{
		map[string]any{"time": "2026-03-06", "close": 100.0, "high": 101, "low": 99, "volume": 1e6},
		map[string]any{"time": "2026-03-09", "close": 101.0, "high": 102, "low": 100, "volume": 1e6},
	}
	signal := map[string]any{"time": time.Date(2026, 3, 9, 10, 0, 0, 0, loc).Format("2006-01-02 15:04:05"), "close": 101.5}
	anchorTime, anchorClose, ok := anchorForSignalBar(signal, daily)
	if !ok || anchorClose != 100.0 {
		t.Fatalf("anchor close=%v ok=%v time=%v", anchorClose, ok, anchorTime)
	}
	if anchorTime.In(loc).Day() != 6 {
		t.Fatalf("expected Friday anchor, got %v", anchorTime.In(loc))
	}
}

func TestPerSignalFilterFourTimeframeRows(t *testing.T) {
	n := 80
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.3
		bars[i] = map[string]any{
			"time": fmtDaily(i), "high": c + 1, "low": c - 1, "close": c, "volume": 1000.0,
		}
	}
	buyM[30] = 1
	daily := make([]any, 40)
	for i := 0; i < 40; i++ {
		c := 100.0 + float64(i)*0.5
		daily[i] = map[string]any{
			"time": fmt.Sprintf("2026-01-%02d", i+1), "close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}
	barsByFreq := map[string][]any{
		"5m": daily, "60m": bars, "daily": daily, "weekly": daily,
	}
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Months: 3,
	}, bars, "5m", buyM, sellM)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("buy triggers=%d", len(stats.BuyTriggers))
	}
	tr := stats.BuyTriggers[0]
	if tr.PriceAnalysis == nil || len(tr.PriceAnalysis.Rows) != 4 {
		t.Fatalf("rows=%d want 4 for 5m gate", len(tr.PriceAnalysis.Rows))
	}
	if tr.AIInterceptConclusion == "" {
		t.Fatal("missing ai_intercept_conclusion")
	}
	if tr.AIInterceptConclusionSource != "rule" {
		t.Fatalf("source=%s", tr.AIInterceptConclusionSource)
	}
	_ = barsByFreq
}

func TestPerSignalFilterEMAFetchSeparateFromPriceAnalysis(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.2
		bars[i] = map[string]any{
			"time": time.Date(2026, 6, 1, 10, 0, 0, 0, loc).AddDate(0, 0, i).Format("2006-01-02 15:04:05"),
			"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
		}
	}
	buyM[10] = 1
	daily := make([]any, 130)
	for i := 0; i < 130; i++ {
		c := 100.0 + float64(i)*0.3
		daily[i] = map[string]any{
			"time": fmt.Sprintf("2026-04-%02d 00:00:00", (i%28)+1),
			"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}
	paMonths := map[string]int{}
	emaMonths := map[string]int{}
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		freq := args["frequency"].(string)
		m := args["months_back"].(int)
		if isEMAStackProbeArgs(args) {
			emaMonths[freq] = m
		} else {
			paMonths[freq] = m
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": daily}}
	})
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "00700.HK", Months: 3, AnnotateOnly: true,
	}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	if paMonths["weekly"] > perSignalPriceFetchMonthsCap {
		t.Fatalf("PA weekly months=%d must stay <= %d", paMonths["weekly"], perSignalPriceFetchMonthsCap)
	}
	if emaMonths["weekly"] < emaWeeklyFetchMonthsFloor {
		t.Fatalf("EMA weekly months=%d want >= %d", emaMonths["weekly"], emaWeeklyFetchMonthsFloor)
	}
	if emaMonths["daily"] < emaDailyFetchMonthsFloor {
		t.Fatalf("EMA daily months=%d want >= %d", emaMonths["daily"], emaDailyFetchMonthsFloor)
	}
}

func TestPerSignalFilterTriggerRecordDiagnoseFields(t *testing.T) {
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.4
		bars[i] = map[string]any{"time": fmtDaily(i), "high": c + 1, "low": c - 1, "close": c, "volume": 1000.0}
	}
	buyM[15] = 1
	daily := make([]any, 130)
	for i := 0; i < 130; i++ {
		c := 100.0 + float64(i)*0.35
		daily[i] = map[string]any{
			"time": fmt.Sprintf("2026-01-%02d", (i%28)+1), "close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
		}
	}
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		minLen := minBarsForPriceFilter(freq) + 5
		b := daily
		if freq == "weekly" || freq == "60m" {
			b = make([]any, minLen)
			for i := 0; i < minLen; i++ {
				c := 100.0 + float64(i)*0.3
				b[i] = map[string]any{
					"time": fmt.Sprintf("2026-05-%02d 10:00:00", (i%28)+1),
					"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
				}
			}
		}
		if isEMAStackProbeArgs(args) {
			b = make([]any, 130)
			for i := 0; i < 130; i++ {
				c := 100.0 + float64(i)*0.35
				b[i] = map[string]any{
					"time": fmt.Sprintf("2026-01-%02d", (i%28)+1), "close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
				}
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": b}}
	})
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "AAPL.US", Months: 3, AnnotateOnly: true, GateScoreMode: PriceAnalysisGateScoreModeHTFOnly,
	}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	tr := stats.BuyTriggers[0]
	if tr.EMAStacks == nil {
		t.Fatal("missing ema_stacks")
	}
	if tr.EMAStacks.Weekly.Label == "" || tr.EMAStacks.Weekly.Label == emaStackLabelUnknown {
		t.Fatalf("weekly ema label=%q want classified stack from EMA fetch", tr.EMAStacks.Weekly.Label)
	}
	if tr.EMAStacks.Daily.Label == emaStackLabelUnknown {
		t.Fatalf("daily ema should classify with extended fetch, got %q", tr.EMAStacks.Daily.Label)
	}
	if tr.GatePriceScore == 0 && tr.PriceScore != 0 {
		// gate may legitimately be 0; ensure field was set by comparing HTF recompute
		htf := ComputeHTFGatePriceScore(*tr.PriceAnalysis, "60m")
		if tr.GatePriceScore != htf.Score {
			t.Fatalf("gate_price_score=%v want htf %v", tr.GatePriceScore, htf.Score)
		}
	}
	if tr.PAWeeklySubscore == nil && tr.TFScores["周线"].Missing {
		// ok when weekly missing
	} else if tr.PAWeeklySubscore == nil {
		t.Fatal("expected pa_weekly_subscore when weekly tf present")
	}
}

func TestPerSignalFilterKeptAndStrippedLists(t *testing.T) {
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i%7)*0.1
		bars[i] = map[string]any{"time": fmtDaily(i), "high": c + 1, "low": c - 1, "close": c, "volume": 1000.0}
	}
	buyM[10] = 1
	buyM[20] = 1
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{Months: 3}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 2 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	if len(stats.KeptBuyTriggers)+len(stats.StrippedBuyTriggers) != 2 {
		t.Fatalf("kept=%d stripped=%d", len(stats.KeptBuyTriggers), len(stats.StrippedBuyTriggers))
	}
}

func fmtDaily(i int) string {
	day := (i % 28) + 1
	return fmt.Sprintf("2026-02-%02d 11:00:00", day)
}

func TestTruncateBarsAtAnchorKeepsBarsOnOrBeforeAnchor(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 1, 15, 16, 0, 0, 0, loc)
	bars := []any{
		map[string]any{"time": "2026-01-10", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0},
		map[string]any{"time": "2026-01-14", "close": 2.0, "high": 2.1, "low": 1.9, "volume": 1.0},
		map[string]any{"time": "2026-01-16", "close": 3.0, "high": 3.1, "low": 2.9, "volume": 1.0},
	}
	trunc := truncateBarsAtAnchor(bars, anchor)
	if len(trunc) != 2 {
		t.Fatalf("trunc len=%d want 2", len(trunc))
	}
}

func TestStructureReasonAfterAnchorWhenWindowStartsAfterAnchor(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 3, 1, 16, 0, 0, 0, loc)
	// Simulates “latest 200 bars from now” missing history before March signal anchor.
	src := []any{
		map[string]any{"time": time.Date(2026, 3, 10, 10, 0, 0, 0, loc).Format(time.RFC3339), "close": 100.0, "high": 101, "low": 99, "volume": 1.0},
	}
	trunc := truncateBarsAtAnchor(src, anchor)
	if len(trunc) != 0 {
		t.Fatalf("expected empty trunc, got %d", len(trunc))
	}
	reason := structureReasonAfterAnchorTruncate(len(src), true)
	if reason != structureNoBarsBeforeAnchor {
		t.Fatalf("reason=%q want %q", reason, structureNoBarsBeforeAnchor)
	}
}

func TestBuildAnchoredPriceAnalysisEmptyStructureMessages(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 2, 1, 16, 0, 0, 0, loc)
	barsByFreq := map[string][]any{
		"5m":  {},
		"60m": {map[string]any{"time": "2026-03-01", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0}},
	}
	fetchOK := map[string]bool{"5m": false, "60m": true}
	pa := buildAnchoredPriceAnalysis(barsByFreq, fetchOK, nil, "60m", anchor, 100, 101, KeyLevelBands{})
	if len(pa.Rows) != 3 {
		t.Fatalf("rows=%d want 3 (60m gate excludes 5m)", len(pa.Rows))
	}
	if pa.Rows[0].Timeframe != "1小时" {
		t.Fatalf("first row=%s want 1小时", pa.Rows[0].Timeframe)
	}
	if pa.Rows[0].Structure != structureNoBarsBeforeAnchor {
		t.Fatalf("60m structure=%q", pa.Rows[0].Structure)
	}
}
