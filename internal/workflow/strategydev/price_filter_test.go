package strategydev

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
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
	bars := []any{
		barOHLCV(0, 101, 99, 100, 1000),
		barOHLCV(1, 101, 99, 101.5, 1000),
		barOHLCV(2, 101, 99, 100, 1000),
		barOHLCV(3, 101, 99, 102, 1000),
	}
	for i, row := range bars {
		m := row.(map[string]any)
		m["time"] = fmt.Sprintf("2026-01-0%d 10:00:00", i+1)
	}
	nb, ns, st := MaskEpisodeTriggers(bars, buy, sell, pf)
	if st.StrippedBuy != 2 || st.StrippedSell != 0 {
		t.Fatalf("stripped buy=%d sell=%d (sell passes when regime unclear)", st.StrippedBuy, st.StrippedSell)
	}
	if st.PreFilterBuyHits != 2 || st.PreFilterSellHits != 1 {
		t.Fatalf("pre-filter buy=%d sell=%d", st.PreFilterBuyHits, st.PreFilterSellHits)
	}
	if st.PostFilterBuyHits != 0 || st.PostFilterSellHits != 1 {
		t.Fatalf("post-filter buy=%d sell=%d", st.PostFilterBuyHits, st.PostFilterSellHits)
	}
	if len(st.StrippedBuyTriggers) != 2 || len(st.StrippedSellTriggers) != 0 {
		t.Fatalf("ledger buy=%d sell=%d", len(st.StrippedBuyTriggers), len(st.StrippedSellTriggers))
	}
	if len(st.BuyTriggers) != 2 || len(st.SellTriggers) != 1 {
		t.Fatalf("all triggers buy=%d sell=%d", len(st.BuyTriggers), len(st.SellTriggers))
	}
	for _, tr := range st.BuyTriggers {
		if !tr.Masked {
			t.Fatal("buy should be masked")
		}
	}
	if !strings.Contains(st.InterceptionBasis, "不明确") {
		t.Fatalf("basis=%q", st.InterceptionBasis)
	}
	for _, v := range nb {
		if int(v.(int)) == 1 {
			t.Fatal("buy should be cleared")
		}
	}
	if int(ns[2].(int)) != -1 {
		t.Fatal("sell should remain when regime is not uptrend")
	}
}

func TestMaskEpisodeTriggersPartialStripKeepsAllowedSide(t *testing.T) {
	pf := PriceFilterResult{Regime: RegimeUp, AllowLong: true, AllowShort: false}
	buy := []any{0, 1, 0}
	sell := []any{0, -1, 0}
	bars := []any{
		barOHLCV(0, 101, 99, 100, 1000),
		barOHLCV(1, 101, 99, 101, 1000),
		barOHLCV(2, 101, 99, 100, 1000),
	}
	nb, ns, st := MaskEpisodeTriggers(bars, buy, sell, pf)
	if st.StrippedBuy != 0 || st.StrippedSell != 1 {
		t.Fatalf("stripped buy=%d sell=%d", st.StrippedBuy, st.StrippedSell)
	}
	if int(nb[1].(int)) != 1 {
		t.Fatal("buy trigger should remain")
	}
	if int(ns[1].(int)) != 0 {
		t.Fatal("sell trigger should be stripped")
	}
}

func TestFinishDiagnoseFromMergedAllStrippedListsLedger(t *testing.T) {
	n := 65
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		row := barOHLCV(i, 101, 99, 100, 1000)
		row["time"] = fmt.Sprintf("2026-03-%02d 11:00:00", (i%28)+1)
		bars[i] = row
	}
	buyM[10] = 1
	buyM[20] = 1
	sellM[15] = -1
	sellM[25] = -1
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		minLen := minBarsForPriceFilter(freq) + 10
		out := make([]any, minLen)
		for j := 0; j < minLen; j++ {
			c := 100.0 + float64(j)*0.2
			out[j] = map[string]any{
				"time": fmt.Sprintf("2026-02-%02d 10:00:00", (j%28)+1),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		if freq == "daily" {
			out = make([]any, 40)
			for j := 0; j < 40; j++ {
				c := 100.0 + float64(j)*0.1
				out[j] = map[string]any{
					"time": fmt.Sprintf("2026-02-%02d", j+1),
					"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
				}
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": out}}
	})
	summary, pf, pstats := finishDiagnoseFromMerged(context.Background(), run, "00700.HK", 3, KeyLevelBands{}, bars, "60m", buyM, sellM, RegimeUp, nil, true, nil)
	if pstats == nil || pf == nil {
		t.Fatal("expected price filter stats")
	}
	if !pf.AllowLong && !pf.AllowShort {
		if pstats.StrippedBuy != 2 || pstats.StrippedSell != 2 {
			t.Fatalf("stripped buy=%d sell=%d", pstats.StrippedBuy, pstats.StrippedSell)
		}
		if len(pstats.StrippedBuyTriggers) != 2 || len(pstats.StrippedSellTriggers) != 2 {
			t.Fatalf("ledger buy=%d sell=%d", len(pstats.StrippedBuyTriggers), len(pstats.StrippedSellTriggers))
		}
		if summary.BuyHits != 0 || summary.SellHits != 0 {
			t.Fatalf("post-filter hits buy=%d sell=%d", summary.BuyHits, summary.SellHits)
		}
		if pstats.PostFilterBuyEpisodes != 0 || pstats.PostFilterSellEpisodes != 0 {
			t.Fatalf("expected zero episodes, buy=%d sell=%d", pstats.PostFilterBuyEpisodes, pstats.PostFilterSellEpisodes)
		}
	}
}

func TestFinishDiagnoseFromMergedSoftAnnotatePreservesHits(t *testing.T) {
	n := 65
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		row := barOHLCV(i, 101, 99, 100, 1000)
		row["time"] = fmt.Sprintf("2026-03-%02d 11:00:00", (i%28)+1)
		bars[i] = row
	}
	buyM[10] = 1
	sellM[15] = -1
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		minLen := minBarsForPriceFilter(freq) + 10
		out := make([]any, minLen)
		for j := 0; j < minLen; j++ {
			c := 100.0 + float64(j)*0.2
			out[j] = map[string]any{
				"time": fmt.Sprintf("2026-02-%02d 10:00:00", (j%28)+1),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		if freq == "daily" {
			out = make([]any, 40)
			for j := 0; j < 40; j++ {
				c := 100.0 + float64(j)*0.1
				out[j] = map[string]any{
					"time": fmt.Sprintf("2026-02-%02d", j+1),
					"close": c, "high": c + 1, "low": c - 1, "volume": 1e6,
				}
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": out}}
	})
	summary, pf, pstats := finishDiagnoseFromMerged(context.Background(), run, "00700.HK", 3, KeyLevelBands{}, bars, "60m", buyM, sellM, RegimeUp, nil, false, nil)
	if pf == nil || pstats == nil {
		t.Fatal("expected soft gate stats")
	}
	if summary.BuyHits != 1 || summary.SellHits != 1 {
		t.Fatalf("hits changed buy=%d sell=%d", summary.BuyHits, summary.SellHits)
	}
	if pstats.PostFilterBuyHits != pstats.PreFilterBuyHits || pstats.PostFilterSellHits != pstats.PreFilterSellHits {
		t.Fatalf("post-filter hits should match pre-filter in soft mode")
	}
	if len(pstats.BuyTriggers) != 1 || pstats.BuyTriggers[0].PriceAnalysis == nil {
		t.Fatal("expected per-trigger price analysis")
	}
	if pstats.BuyTriggers[0].Masked {
		t.Fatal("soft gate must not mask")
	}
}

func TestUsePriceAnalysisEpisodeFilterDefaultOff(t *testing.T) {
	if UsePriceAnalysisEpisodeFilterEnabled(nil) {
		t.Fatal("nil should default to disabled (soft gate)")
	}
	on := true
	if !UsePriceAnalysisEpisodeFilterEnabled(&on) {
		t.Fatal("explicit true should enable hard mask")
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
