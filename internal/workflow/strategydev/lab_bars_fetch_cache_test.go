package strategydev

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
)

type countingBarsFetcher struct {
	calls atomic.Int32
}

func (f *countingBarsFetcher) FetchBars(_ context.Context, _, _ string, _ int, asOf *time.Time) ([]any, bool) {
	if asOf != nil {
		panic("cache miss should not pass as_of to lab-api")
	}
	f.calls.Add(1)
	return []any{map[string]any{"time": "2026-10-10 00:00:00", "close": 420.0}}, true
}

func TestCachedLabBarsFetcherReusesSameRunKey(t *testing.T) {
	inner := &countingBarsFetcher{}
	cached := WrapCachedLabBarsFetcher("sess", "run-1", inner).(CachedLabBarsFetcher)
	ctx := context.Background()
	_, _ = cached.FetchBars(ctx, "00700.HK", "5m", 4, nil)
	_, _ = cached.FetchBars(ctx, "00700.HK", "5m", 4, nil)
	if inner.calls.Load() != 1 {
		t.Fatalf("want 1 lab-api fetch, got %d", inner.calls.Load())
	}
	otherRun := WrapCachedLabBarsFetcher("sess", "run-2", inner).(CachedLabBarsFetcher)
	_, _ = otherRun.FetchBars(ctx, "00700.HK", "5m", 4, nil)
	if inner.calls.Load() != 2 {
		t.Fatalf("different run_id should miss cache, got %d calls", inner.calls.Load())
	}
}

func TestBuildLabEpisodeDetailReusesPAFetchAcrossEpisodes(t *testing.T) {
	inner := &countingBarsFetcher{}
	bars := []any{
		map[string]any{"time": "2026-07-22 00:00:00", "close": 430.0},
		map[string]any{"time": "2026-07-24 11:30:00", "close": 436.0},
	}
	wrapped := WrapCachedLabBarsFetcher("sess", "run-pa", inner)
	in := LabDetailInput{
		Code: "00700.HK", SignalFreq: "5m", Months: 3, Bars: bars,
		LabFetcher: wrapped, SessionID: "sess", RunID: "run-pa",
		DetailTimeout: 0,
	}
	if _, err := BuildLabEpisodeDetail(context.Background(), in, LabDetailRef("buy", 0)); err != nil {
		t.Fatal(err)
	}
	first := inner.calls.Load()
	if first == 0 {
		t.Fatal("expected lab fetches on first episode")
	}
	if _, err := BuildLabEpisodeDetail(context.Background(), in, LabDetailRef("buy", 1)); err != nil {
		t.Fatal(err)
	}
	if inner.calls.Load() != first {
		t.Fatalf("second episode should reuse cached bars: before=%d after=%d", first, inner.calls.Load())
	}
}

func TestRunAnchoredTruncateAfterFullFetch(t *testing.T) {
	full := []any{
		map[string]any{"time": "2026-07-20 10:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0},
		map[string]any{"time": "2026-10-10 10:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0},
	}
	fetch := fullWindowBarsFetcher{bars: full, t: t}
	anchor := parseTestTime(t, "2026-07-24 11:30:00")
	pa := priceanalysis.Run(context.Background(), fetch, priceanalysis.RunRequest{
		Code: "00700.HK", Months: 3, SignalFreq: "60m", AsOf: &anchor, SpotPrice: 436,
	})
	for _, row := range pa.Rows {
		if row.Structure == "拉数失败" {
			t.Fatalf("unexpected fetch failure row: %+v", row)
		}
	}
}

type fullWindowBarsFetcher struct {
	bars []any
	t    *testing.T
}

func (f fullWindowBarsFetcher) FetchBars(_ context.Context, _, _ string, _ int, asOf *time.Time) ([]any, bool) {
	if asOf != nil {
		f.t.Fatal("unexpected as_of")
	}
	return f.bars, true
}

func parseTestTime(t *testing.T, s string) (out time.Time) {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	out, err = time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
