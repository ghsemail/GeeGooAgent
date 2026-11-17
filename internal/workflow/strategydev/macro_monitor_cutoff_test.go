package strategydev

import (
	"testing"
	"time"
)

func TestResolveMacroMonitorCutoff_RejectsIntradayIncomplete_CN(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 11, 16, 10, 30, 0, 0, loc)
	bars := []any{
		map[string]any{"time": "2026-11-14 00:00:00", "close": 3920.5},
		map[string]any{"time": "2026-11-16 10:30:00", "close": 3990.0},
	}
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("000300.SH", now, bars)
	if !ok {
		t.Fatal("expected completed session from 11-14")
	}
	if cut.CalendarDay != "2026-11-14" {
		t.Fatalf("day=%q want 2026-11-14", cut.CalendarDay)
	}
	if cut.Close != 3920.5 {
		t.Fatalf("close=%v", cut.Close)
	}
}

func TestResolveMacroMonitorCutoff_UsesSameDayAfterClose_CN(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 11, 16, 16, 30, 0, 0, loc)
	bars := []any{
		map[string]any{"time": "2026-11-14 00:00:00", "close": 3920.5},
		map[string]any{"time": "2026-11-16 00:00:00", "close": 3979.8},
	}
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("000300.SH", now, bars)
	if !ok {
		t.Fatal("expected ok")
	}
	if cut.CalendarDay != "2026-11-16" {
		t.Fatalf("day=%q want 2026-11-16", cut.CalendarDay)
	}
}

func TestResolveMacroMonitorCutoff_FailClosedUnknownMarket(t *testing.T) {
	_, ok := resolveMacroMonitorCutoffFromDailyBars("FOO.BAR", time.Now(), []any{
		map[string]any{"time": "2026-11-14 00:00:00", "close": 1.0},
	})
	if ok {
		t.Fatal("expected fail-closed for unknown suffix")
	}
}

func TestResolveMacroMonitorCutoff_SevenPoolSuffixFixtures(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	afterCN := time.Date(2026, 11, 16, 16, 0, 0, 0, loc)
	bar := []any{map[string]any{"time": "2026-11-16 00:00:00", "close": 1.0}}
	locHK, _ := time.LoadLocation("Asia/Hong_Kong")
	afterHK := time.Date(2026, 11, 16, 16, 0, 0, 0, locHK)
	cases := []struct {
		code string
		when time.Time
	}{
		{"000300.SH", afterCN},
		{"00700.HK", afterHK},
		{"SPY.US", time.Date(2026, 11, 16, 22, 0, 0, 0, time.UTC)},
		{"BTC-USDT.CRYPTO", time.Date(2026, 11, 16, 23, 58, 0, 0, time.UTC)},
		{"DX-Y.NYB.US", time.Date(2026, 11, 16, 22, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		_, ok := resolveMacroMonitorCutoffFromDailyBars(tc.code, tc.when, bar)
		if !ok {
			t.Fatalf("code=%s expected ok for fixture completed bar", tc.code)
		}
	}
}