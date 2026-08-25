package tools

import (
	"testing"
	"time"
)

func TestPremarketAlreadyReportedForSessionUS(t *testing.T) {
	t.Parallel()
	loc := time.Local
	stale := time.Date(2026, 8, 24, 21, 13, 0, 0, loc).UTC().Format(time.RFC3339)
	reports := []map[string]any{{"created_at": stale}}

	if premarketAlreadyReportedForSession("SPCX.US", "2026-08-25", reports) {
		t.Fatal("expected stale US evening report not to satisfy next session")
	}
	if !premarketAlreadyReportedForSession("SPCX.US", "2026-08-24", reports) {
		t.Fatal("expected same-session US report to satisfy idempotency")
	}
}

func TestPremarketAlreadyReportedForSessionUSDaytimeBackfill(t *testing.T) {
	t.Parallel()
	loc := time.Local
	// Manual backfill created early morning on session day must not block 21:10 cron.
	backfill := time.Date(2026, 8, 26, 1, 24, 46, 0, loc).UTC().Format(time.RFC3339)
	reports := []map[string]any{{"created_at": backfill}}
	if premarketAlreadyReportedForSession("SPCX.US", "2026-08-26", reports) {
		t.Fatal("expected daytime backfill not to block evening premarket run")
	}

	evening := time.Date(2026, 8, 26, 21, 15, 0, 0, loc).UTC().Format(time.RFC3339)
	reports = []map[string]any{{"created_at": evening}}
	if !premarketAlreadyReportedForSession("SPCX.US", "2026-08-26", reports) {
		t.Fatal("expected evening US report to satisfy idempotency")
	}
}

func TestPremarketAlreadyReportedForSessionCNHK(t *testing.T) {
	t.Parallel()
	reports := []map[string]any{{"created_at": "2026-08-25T01:15:23.217Z"}}
	if !premarketAlreadyReportedForSession("00700.HK", "2026-08-25", reports) {
		t.Fatal("expected CN/HK rows to use count-based idempotency")
	}
}
