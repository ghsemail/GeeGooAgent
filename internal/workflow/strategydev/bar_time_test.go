package strategydev

import (
	"testing"
	"time"
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
