package strategydev

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
)

type staticFailBarsFetcher struct{}

func (staticFailBarsFetcher) FetchBars(context.Context, string, string, int, *time.Time) ([]any, bool) {
	return nil, false
}

type coarseFailBarsFetcher struct{}

func (coarseFailBarsFetcher) FetchBars(_ context.Context, _, freq string, _ int, _ *time.Time) ([]any, bool) {
	if freq == "daily" || freq == "weekly" {
		return nil, false
	}
	return synthTrendBars(130), true
}

type failingThenOKFetcher struct {
	calls atomic.Int32
}

func (f *failingThenOKFetcher) FetchBars(_ context.Context, _, freq string, _ int, _ *time.Time) ([]any, bool) {
	f.calls.Add(1)
	if freq == "daily" && f.calls.Load() == 1 {
		return nil, false
	}
	return []any{map[string]any{
		"time": "2026-08-01 00:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0,
	}}, true
}

func synthTrendBars(n int) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.1
		out[i] = map[string]any{
			"time": "2026-06-01 10:00:00", "open": c, "high": c + 1, "low": c - 1, "close": c, "volume": 1000.0,
		}
	}
	return out
}

func TestCachedLabBarsFetcherSkipsNegativeCache(t *testing.T) {
	inner := &failingThenOKFetcher{}
	cached := WrapCachedLabBarsFetcher("sess", "run-neg", inner)
	ctx := context.Background()
	months, err := priceanalysis.MonthsBackForLabFetch(3, "daily")
	if err != nil {
		t.Fatal(err)
	}
	_, ok1 := cached.FetchBars(ctx, "AAPL.US", "daily", months, nil)
	if ok1 {
		t.Fatal("first fetch should fail")
	}
	_, ok2 := cached.FetchBars(ctx, "AAPL.US", "daily", months, nil)
	if !ok2 {
		t.Fatal("second fetch must retry inner, not reuse negative cache")
	}
	if inner.calls.Load() != 2 {
		t.Fatalf("want 2 inner calls, got %d", inner.calls.Load())
	}
}

func TestMTFFallbackBarsFetcherUsesProbeDailyWeekly(t *testing.T) {
	mtf := &KlinePatternMTFBundle{
		ByFreq: map[string]KlinePatternFreqSeries{
			"daily":  {Freq: "daily", FetchOK: true, Bars: synthTrendBars(80)},
			"weekly": {Freq: "weekly", FetchOK: true, Bars: synthTrendBars(30)},
		},
	}
	inner := staticFailBarsFetcher{}
	fb := wrapMTFFallbackBarsFetcher(inner, mtf)
	dBars, dOK := fb.FetchBars(context.Background(), "AAPL.US", "daily", 7, nil)
	if !dOK || len(dBars) == 0 {
		t.Fatalf("daily fallback: ok=%v len=%d", dOK, len(dBars))
	}
	wBars, wOK := fb.FetchBars(context.Background(), "AAPL.US", "weekly", 10, nil)
	if !wOK || len(wBars) == 0 {
		t.Fatalf("weekly fallback: ok=%v len=%d", wOK, len(wBars))
	}
}

func TestBuildLabEpisodeDetailDailyWeeklyNotFetchFailedWithMTFFallback(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	mtfDaily := make([]any, 80)
	for i := range mtfDaily {
		day := 1 + i
		mtfDaily[i] = map[string]any{
			"time": time.Date(2026, 5, day, 0, 0, 0, 0, loc).Format("2006-01-02 15:04:05"),
			"open": 200.0, "high": 201.0, "low": 199.0, "close": 200.0 + float64(i)*0.1, "volume": 1e6,
		}
	}
	mtfWeekly := make([]any, 30)
	for i := range mtfWeekly {
		mtfWeekly[i] = map[string]any{
			"time": time.Date(2026, 1, 5, 0, 0, 0, 0, loc).AddDate(0, 0, i*7).Format("2006-01-02 15:04:05"),
			"open": 180.0, "high": 181.0, "low": 179.0, "close": 180.0 + float64(i)*0.2, "volume": 1e6,
		}
	}
	mtf := &KlinePatternMTFBundle{
		SignalFreq: "60m",
		ByFreq: map[string]KlinePatternFreqSeries{
			"60m":    {Freq: "60m", FetchOK: true, Bars: synthTrendBars(130)},
			"daily":  {Freq: "daily", FetchOK: true, Bars: mtfDaily},
			"weekly": {Freq: "weekly", FetchOK: true, Bars: mtfWeekly},
		},
	}
	inner := coarseFailBarsFetcher{}
	bars := make([]any, 78)
	for i := range bars {
		bars[i] = map[string]any{
			"time": time.Date(2026, 6, 1, 10, 0, 0, 0, loc).Add(time.Duration(i) * time.Hour).Format("2006-01-02 15:04:05"),
			"close": 300.0 + float64(i),
		}
	}
	cachedMTF := mtf
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "AAPL.US", SignalFreq: "60m", Months: 3, Bars: bars,
		LabFetcher: WrapCachedLabBarsFetcher("sess", "run-aapl", inner),
		CachedMTF:  &cachedMTF, DetailTimeout: 0, PATimeout: 0,
	}, LabDetailRef("sell", 77))
	if err != nil {
		t.Fatal(err)
	}
	if detail.PriceAnalysis == nil {
		t.Fatal("expected price analysis")
	}
	for _, row := range detail.PriceAnalysis.Rows {
		if row.Timeframe != "日线" && row.Timeframe != "周线" {
			continue
		}
		if row.Structure == "拉数失败" {
			t.Fatalf("%s should not be 拉数失败: %+v", row.Timeframe, row)
		}
	}
}
