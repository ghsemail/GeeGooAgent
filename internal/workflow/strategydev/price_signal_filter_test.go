package strategydev

import (
	"context"
	"fmt"
	"testing"
	"time"
)

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
	}, bars, "60m", buyM, sellM)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("buy triggers=%d", len(stats.BuyTriggers))
	}
	tr := stats.BuyTriggers[0]
	if tr.PriceAnalysis == nil || len(tr.PriceAnalysis.Rows) != 4 {
		t.Fatalf("rows=%d", len(tr.PriceAnalysis.Rows))
	}
	if tr.AIInterceptConclusion == "" {
		t.Fatal("missing ai_intercept_conclusion")
	}
	if tr.AIInterceptConclusionSource != "rule" {
		t.Fatalf("source=%s", tr.AIInterceptConclusionSource)
	}
	_ = barsByFreq
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
