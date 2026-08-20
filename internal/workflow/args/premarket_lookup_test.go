package args_test

import (
	"testing"

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
	if got := args.PremarketReportDateForPostmarket("SPCX.US", "2026-08-20"); got != "2026-08-19" {
		t.Fatalf("got %q want 2026-08-19", got)
	}
}
