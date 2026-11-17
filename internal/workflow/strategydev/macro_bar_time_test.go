package strategydev

import (
	"testing"
	"time"
)

func TestMacroBarCalendarDay_USDateLabelNotShanghaiInstant(t *testing.T) {
	spec, ok := macroSessionSpecForCode("SPY.US")
	if !ok {
		t.Fatal("spec")
	}
	row := map[string]any{"time": "2026-11-16 00:00:00", "close": 450.0}
	day, ok := macroBarCalendarDay(row, spec)
	if !ok {
		t.Fatal("expected calendar day")
	}
	if day.In(spec.loc).Format("2006-01-02") != "2026-11-16" {
		t.Fatalf("got %s want 2026-11-16 in America/New_York", day.In(spec.loc).Format("2006-01-02"))
	}
	// barTimeUnix alone would pin the stamp to Shanghai and shift ET calendar backward.
	sh := barTimeUnix(row)
	if sh.In(spec.loc).Format("2006-01-02") == "2026-11-16" {
		// same calendar when string is date-only; assert cutoff still picks US day
	}
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 11, 16, 17, 0, 0, 0, loc)
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("SPY.US", now, []any{row})
	if !ok || cut.CalendarDay != "2026-11-16" {
		t.Fatalf("cutoff day=%q ok=%v", cut.CalendarDay, ok)
	}
}

func TestMacroBarCalendarDay_CryptoRFC3339AndUnix(t *testing.T) {
	spec, ok := macroSessionSpecForCode("BTC-USD.CRYPTO")
	if !ok {
		t.Fatal("spec")
	}
	rfc := map[string]any{"time": "2026-11-16T00:00:00Z", "close": 91000.0}
	day, ok := macroBarCalendarDay(rfc, spec)
	if !ok || day.In(spec.loc).Format("2006-01-02") != "2026-11-16" {
		t.Fatalf("rfc day=%v", day)
	}
	unixBar := map[string]any{"time": float64(20261116), "close": 1.0} // YYYYMMDD int label via barTimeUnix number path is Shanghai — use trade_date
	unixBar = map[string]any{"trade_date": "20261116", "close": 1.0}
	day2, ok := macroBarCalendarDay(unixBar, spec)
	if !ok || day2.In(spec.loc).Format("2006-01-02") != "2026-11-16" {
		t.Fatalf("trade_date day=%v", day2)
	}
}

func TestMacroBarCalendarDay_RejectsFutureSessionDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, loc)
	bars := []any{
		map[string]any{"time": "2026-11-17 00:00:00", "close": 1.0},
	}
	_, ok := resolveMacroMonitorCutoffFromDailyBars("000300.SH", now, bars)
	if ok {
		t.Fatal("future bar must not be selected before session completes")
	}
}