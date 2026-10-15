package strategydev

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestBarTimeUnixAShareTradeDateSlash(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"trade_date": "2026/03/09 15:00"}
	ts := barTimeUnix(row)
	want := time.Date(2026, 3, 9, 15, 0, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestBarTimeUnixAShareYYYYMMDDInt(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"date": float64(20260309)}
	ts := barTimeUnix(row)
	want := time.Date(2026, 3, 9, 0, 0, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestBarTimeUnixCN60mCompactWithSuffix(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"time": "202604291130 00:00:00"}
	ts := barTimeUnix(row)
	want := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestBarTimeUnixCN60mCompactNumeric(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"time": float64(202604291130)}
	ts := barTimeUnix(row)
	want := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestAnchorForSignalAShareDailyTradeDate300750(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	daily := []any{
		map[string]any{"trade_date": "2026/03/06 15:00", "close": 248.5, "high": 250, "low": 247, "volume": 1e6},
		map[string]any{"trade_date": "2026/03/09 15:00", "close": 251.2, "high": 252, "low": 249, "volume": 1e6},
	}
	signal := map[string]any{"trade_date": "2026/03/09 11:30", "close": 250.0, "high": 251, "low": 249, "volume": 1e5}
	anchorTime, anchorClose, ok := anchorForSignalBar(signal, daily)
	if !ok {
		t.Fatal("expected prior trading day anchor for A-share trade_date bars")
	}
	if anchorClose != 248.5 {
		t.Fatalf("anchor close=%v want 248.5 (2026-03-06)", anchorClose)
	}
	wantAnchor := time.Date(2026, 3, 6, 16, 0, 0, 0, loc)
	if !anchorTime.Equal(wantAnchor) {
		t.Fatalf("anchor instant=%v want %v", anchorTime.In(loc), wantAnchor)
	}
}

func TestPerSignalFilterCNCompact60mTriggerTime300750(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 20
	bars := make([]any, n)
	buyM := make([]any, n)
	for i := 0; i < n; i++ {
		bars[i] = map[string]any{
			"time":   "202604291130 00:00:00",
			"close":  250.0 + float64(i)*0.1,
			"high":   251.0,
			"low":    249.0,
			"volume": 1e5,
		}
	}
	buyM[5] = 1
	daily := []any{
		map[string]any{"trade_date": "2026/04/28 15:00", "close": 248.0, "high": 249, "low": 247, "volume": 1e6},
		map[string]any{"trade_date": "2026/04/29 15:00", "close": 250.5, "high": 251, "low": 249, "volume": 1e6},
	}
	minLen := minBarsForPriceFilter("60m") + 5
	mkIntra := func() []any {
		out := make([]any, minLen)
		for j := 0; j < minLen; j++ {
			c := 240.0 + float64(j)*0.05
			out[j] = map[string]any{
				"time":   fmt.Sprintf("2026-04-%02d 10:00:00", (j%20)+1),
				"close":  c,
				"high":   c + 1,
				"low":    c - 1,
				"volume": 1000.0,
			}
		}
		return out
	}
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			return tools.Result{Status: tools.StatusError}
		}
		freq := args["frequency"].(string)
		var b []any
		switch freq {
		case "daily":
			b = daily
		default:
			b = mkIntra()
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": b}}
	})
	_, _, _, stats := ApplyPerSignalPriceAnalysisEpisodeFilter(context.Background(), PerSignalPriceFilterInput{
		Run: run, Code: "300750.SZ", Months: 3,
	}, bars, "60m", buyM, nil)
	if len(stats.BuyTriggers) != 1 {
		t.Fatalf("triggers=%d", len(stats.BuyTriggers))
	}
	tr := stats.BuyTriggers[0]
	if tr.PriceAnalysis == nil {
		t.Fatal("missing price analysis")
	}
	if tr.PriceAnalysis.SpotLabel == structurePriorTradingDayUnavailable {
		t.Fatalf("anchor failed: %+v", tr.PriceAnalysis)
	}
	if tr.PriceAnalysis.AnchorClose != 248.0 {
		t.Fatalf("anchor close=%v want 248 (2026-04-28)", tr.PriceAnalysis.AnchorClose)
	}
	if tr.PriceScore == 0 && tr.ScoreReason == "" {
		t.Fatal("expected scored trigger")
	}
	if strings.Contains(tr.AIInterceptConclusion, "已拦截") && !tr.WouldIntercept {
		t.Fatalf("soft kept trigger must not say 已拦截: %q", tr.AIInterceptConclusion)
	}
	_ = loc
}
