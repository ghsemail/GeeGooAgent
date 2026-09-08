package args_test

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/args"
)

func TestPremarketReportDateForPostmarketCNHK(t *testing.T) {
	t.Parallel()
	if got := args.PremarketReportDateForPostmarket("601766.SH", "2026-08-20"); got != "2026-08-20" {
		t.Fatalf("got %q", got)
	}
	if got := args.PremarketReportDateForPostmarket("00700.HK", "2026-08-20"); got != "2026-08-20" {
		t.Fatalf("got %q", got)
	}
}

func TestPremarketReportDateForPostmarketUS(t *testing.T) {
	t.Parallel()
	if got := args.PremarketReportDateForPostmarket("SPCX.US", "2026-08-20"); got != "2026-08-20" {
		t.Fatalf("got %q want 2026-08-20", got)
	}
}

func TestTradingDayProbeCode(t *testing.T) {
	t.Parallel()
	if got := args.TradingDayProbeCode("US"); got != "AAPL.US" {
		t.Fatalf("got %q", got)
	}
	if got := args.TradingDayProbeCode("CN"); got != "000001.SZ" {
		t.Fatalf("got %q", got)
	}
}

func TestCalendarDateForMarketUSUsesEastern(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// Tuesday 05:00 Asia/Shanghai = Monday 17:00 EDT
	now := time.Date(2026, 9, 8, 5, 0, 0, 0, time.FixedZone("CST", 8*3600))
	got := args.CalendarDateForMarket("US", now)
	want := now.In(loc).Format("2006-01-02")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got != "2026-09-07" {
		t.Fatalf("expected US session 2026-09-07, got %q", got)
	}
}
