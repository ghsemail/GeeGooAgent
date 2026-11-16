package strategydev

import (
	"testing"
	"time"
)

func TestAnchorForDailyMacroMonitorUsesNewestDailyBar(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2026-11-04 00:00:00", "close": 4101.34},
		map[string]any{"time": "2026-11-13 15:00:00", "close": 3979.80},
		map[string]any{"time": "2026-11-14 00:00:00", "close": 3920.5},
	}
	at, closePx, ok := anchorForDailyMacroMonitor(bars)
	if !ok {
		t.Fatal("expected ok")
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	want := time.Date(2026, 11, 14, 16, 0, 0, 0, loc)
	if !at.Equal(want) {
		t.Fatalf("anchor=%v want %v", at.In(loc), want)
	}
	if closePx != 3920.5 {
		t.Fatalf("close=%v", closePx)
	}
}