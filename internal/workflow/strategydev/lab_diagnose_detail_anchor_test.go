package strategydev

import (
	"context"
	"slices"
	"testing"
	"time"
)

type monthsRecordingDailyFetcher struct {
	lastMonths int
}

func (f *monthsRecordingDailyFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	if freq != "daily" {
		return nil, true
	}
	f.lastMonths = months
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// Short window (UI months_back=3): omits prior trading day before Jul 24 signal.
	if months <= 3 {
		return []any{
			map[string]any{"time": "2026-08-01 00:00:00", "close": 440.0},
			map[string]any{"time": "2026-10-10 00:00:00", "close": 450.0},
		}, true
	}
	_ = loc
	return []any{
		map[string]any{"time": "2026-07-23 00:00:00", "close": 432.60},
		map[string]any{"time": "2026-07-24 00:00:00", "close": 435.0},
		map[string]any{"time": "2026-10-10 00:00:00", "close": 450.0},
	}, true
}

func TestBuildLabEpisodeDetailExtendsDailyFetchForPriorTradingDayAnchor(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	lookback := []any{
		map[string]any{"time": "2026-06-24 00:00:00", "close": 420.0},
		map[string]any{"time": "2026-10-10 00:00:00", "close": 450.0},
	}
	bars := []any{
		map[string]any{"time": "2026-07-22 00:00:00", "close": 430.0},
		map[string]any{"time": "2026-07-23 00:00:00", "close": 431.0},
		map[string]any{"time": "2026-07-24 11:30:00", "close": 436.0},
	}
	fetcher := &monthsRecordingDailyFetcher{}
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "daily", Months: 3,
		Bars: bars, LookbackBars: lookback, LabFetcher: fetcher,
	}, LabDetailRef("sell", 2))
	if err != nil {
		t.Fatal(err)
	}
	if fetcher.lastMonths <= 3 {
		t.Fatalf("daily fetch months=%d want extended span from chart lookback", fetcher.lastMonths)
	}
	if slices.Contains(detail.Degraded, "price_anchor_missing") {
		t.Fatalf("unexpected price_anchor_missing degraded=%v", detail.Degraded)
	}
	if detail.PriceAnalysis == nil {
		t.Fatal("expected price analysis")
	}
	if detail.PriceAnalysis.AnchorTime == "" {
		t.Fatal("expected anchored price analysis as_of from prior trading day")
	}
	wantAnchor := time.Date(2026, 7, 23, 16, 0, 0, 0, loc)
	row := barRowAt(bars, 2)
	at, close, ok := anchorForSignalBar(row, dailyBarsForLabAnchor(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "daily", Months: 3,
		Bars: bars, LookbackBars: lookback, LabFetcher: fetcher,
	}))
	if !ok || close != 432.60 || !at.Equal(wantAnchor) {
		t.Fatalf("anchor resolve close=%v ok=%v time=%v", close, ok, at.In(loc))
	}
}
