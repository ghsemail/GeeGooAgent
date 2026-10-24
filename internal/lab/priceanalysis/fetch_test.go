package priceanalysis

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchMultiTimeframeBarsDoesNotPassAsOfToFetcher(t *testing.T) {
	anchor := time.Date(2026, 7, 24, 11, 30, 0, 0, time.UTC)
	var calls atomic.Int32
	fetch := BarsFetcherFunc(func(_ context.Context, _, freq string, _ int, asOf *time.Time) ([]any, bool) {
		calls.Add(1)
		if asOf != nil {
			t.Fatalf("freq=%s unexpected as_of on lab fetch", freq)
		}
		return []any{map[string]any{"time": "2026-10-10 00:00:00", "close": 100.0}}, true
	})
	_, _, _ = FetchMultiTimeframeBars(context.Background(), MultiTFBarsInput{
		Fetch: fetch, Code: "00700.HK", Months: 3, GateFreq: "5m", AsOf: &anchor,
	})
	if calls.Load() < 2 {
		t.Fatalf("expected multi-TF fetches, got %d", calls.Load())
	}
}
