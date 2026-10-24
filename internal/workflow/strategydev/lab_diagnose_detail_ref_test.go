package strategydev

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// stubLabBarsOK returns minimal OHLCV for any TF (lab-detail modal happy path).
type stubLabBarsOK struct{}

func (stubLabBarsOK) FetchBars(_ context.Context, _, _ string, _ int, _ *time.Time) ([]any, bool) {
	return []any{
		map[string]any{"time": "2026-07-20 10:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0},
		map[string]any{"time": "2026-10-10 10:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0},
	}, true
}

func TestBuildLabEpisodeDetail_WrongRefInRangeNot400(t *testing.T) {
	n := 200
	bars := make([]any, n)
	for i := range bars {
		bars[i] = map[string]any{"time": "2026-07-20 10:00:00", "close": float64(400 + i)}
	}
	bars[7] = map[string]any{"time": "2026-06-01 10:00:00", "close": 407.0}
	bars[190] = map[string]any{"time": "2026-07-24 11:30:00", "close": 436.0}

	mtf := &KlinePatternMTFBundle{
		SignalFreq: "60m",
		ByFreq: map[string]KlinePatternFreqSeries{
			"daily":  {Freq: "daily", FetchOK: true, Bars: bars[:10]},
			"weekly": {Freq: "weekly", FetchOK: true, Bars: bars[:5]},
		},
	}
	cached := mtf
	in := LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 3, Bars: bars,
		LabFetcher: stubLabBarsOK{}, DetailTimeout: 0, CachedMTF: &cached,
	}

	wrong, err := BuildLabEpisodeDetail(context.Background(), in, "sell:7")
	if err != nil {
		t.Fatalf("sell:7 in range must not 400: %v", err)
	}
	if wrong.BarIndex != 7 || wrong.Ref != "sell:7" || wrong.Close != 407.0 {
		t.Fatalf("wrong=%+v", wrong)
	}

	right, err := BuildLabEpisodeDetail(context.Background(), in, "sell:190")
	if err != nil {
		t.Fatal(err)
	}
	if right.Close != 436.0 {
		t.Fatalf("sell:190 close=%v", right.Close)
	}
	if wrong.Close == right.Close {
		t.Fatal("sell:7 and sell:190 must differ")
	}

	_, err = BuildLabEpisodeDetail(context.Background(), in, "sell:99999")
	if err == nil || !IsBadLabRef(err) {
		t.Fatalf("out of range want bad ref, err=%v", err)
	}
}

func TestLabDetailRefMismatchHint_EpisodeIndexVsStartIdx(t *testing.T) {
	ep7, ep3 := 7, 3
	sums := LabCardEpisodeSummaries{
		"sell:190": {DetailRef: "sell:190", StartIdx: 190, EpisodeIndex: &ep7},
		"buy:7":    {DetailRef: "buy:7", StartIdx: 7, EpisodeIndex: &ep3},
	}
	hint := LabDetailRefMismatchHint(sums, "sell:7")
	if hint == "" || !strings.Contains(hint, "sell:190") {
		t.Fatalf("hint=%q", hint)
	}
	if LabDetailRefMismatchHint(sums, "sell:190") != "" {
		t.Fatal("canonical ref should not hint")
	}
}

func TestBuildLabEpisodeDetail_NoFetchFailureStringsWhenBarsOK(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2026-07-24 11:30:00", "close": 436.0},
	}
	mtf := &KlinePatternMTFBundle{
		SignalFreq: "60m",
		ByFreq: map[string]KlinePatternFreqSeries{
			"daily":  {Freq: "daily", FetchOK: true, Bars: bars},
			"weekly": {Freq: "weekly", FetchOK: true, Bars: bars},
		},
	}
	cached := mtf
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 3, Bars: bars,
		LabFetcher: stubLabBarsOK{}, DetailTimeout: 0, CachedMTF: &cached,
	}, "sell:0")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(detail)
	s := string(raw)
	for _, bad := range []string{"拉数失败", "取数超时或已取消"} {
		if strings.Contains(s, bad) {
			t.Fatalf("unexpected %q in detail json: %s", bad, s)
		}
	}
}
