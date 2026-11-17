package strategydev

import (
	"testing"
	"time"
)

func TestMacroBarVendorLabel_DateOnlyWithoutTZ_NoShanghaiShift(t *testing.T) {
	spec, ok := macroSessionSpecForCode("BTC-USDT.CRYPTO")
	if !ok || !spec.cryptoVendorLabel {
		t.Fatal("crypto vendor spec")
	}
	row := map[string]any{"time": "2026-11-16 00:00:00", "close": 91000.0, "open": 1, "high": 2, "low": 0.5}
	day, ok := macroBarCalendarDay(row, spec)
	if !ok || day.Format("2006-01-02") != "2026-11-16" {
		t.Fatalf("label=%v", day)
	}
	sh := barTimeUnix(row)
	if sh.In(mustShanghai(t)).Format("2006-01-02") != "2026-11-16" {
		// If Shanghai path differs, vendor label must still be 2026-11-16 per contract C.
		if day.Format("2006-01-02") != "2026-11-16" {
			t.Fatalf("vendor label must not follow Shanghai shift")
		}
	}
}

func TestResolveMacroMonitorCutoff_Crypto_NotGatedByUTC2359(t *testing.T) {
	bar := map[string]any{
		"time": "2026-11-16 00:00:00", "open": 1.0, "high": 2.0, "low": 0.9, "close": 1.5,
	}
	now := time.Date(2026, 11, 16, 23, 58, 0, 0, time.UTC)
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("BTC-USDT.CRYPTO", now, []any{bar})
	if !ok || cut.CalendarDay != "2026-11-16" {
		t.Fatalf("must accept vendor label day before UTC 23:59 gate; ok=%v day=%q", ok, cut.CalendarDay)
	}
}

func TestResolveMacroMonitorCutoff_Crypto_CheckTradingDayDateIsLabel(t *testing.T) {
	bar := map[string]any{"time": "2026-11-16 00:00:00", "open": 1, "high": 1, "low": 1, "close": 2.0}
	spec, _ := macroSessionSpecForCode("BTC-USDT.CRYPTO")
	day, _ := macroBarCalendarDay(bar, spec)
	if day.Format("2006-01-02") != "2026-11-16" {
		t.Fatal("label day")
	}
}

func mustShanghai(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}