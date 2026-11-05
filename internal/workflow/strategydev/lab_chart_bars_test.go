package strategydev

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// Simulates 00700 60m 3mo: chart_probe has clamped bars; lab-detail mistakenly indexed lookback.
func TestBuildLabEpisodeDetail_AnchorMatchesChartProbeNotLookbackIndex(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	windowEnd := time.Date(2026, 11, 5, 10, 30, 0, 0, loc)

	var lookback []any
	day := time.Date(2026, 5, 5, 10, 30, 0, 0, loc)
	for i := 0; !day.After(windowEnd); i++ {
		lookback = append(lookback, map[string]any{
			"time": day.Format("2006-01-02 15:04:05"), "close": 300.0 + float64(i),
		})
		day = day.Add(60 * time.Minute)
	}
	probeRaw := map[string]any{"bars": append([]any(nil), lookback...)}
	ClampProbeSeriesToMonthsBack(probeRaw, 3)
	chartBars, _ := probeRaw["bars"].([]any)
	if len(chartBars) == 0 {
		t.Fatal("expected clamped chart bars")
	}
	signalIdx := len(chartBars) - 3
	wantTime := barTimeString(chartBars[signalIdx].(map[string]any))
	wrongTime := barTimeString(lookback[signalIdx].(map[string]any))
	if wantTime == wrongTime {
		t.Fatalf("fixture need lookback/chart mismatch at idx %d", signalIdx)
	}

	eval := slots.EvaluateSignalEpisodes(chartBars, nil, nil)
	eval.BuyDetails = []slots.SignalEpisodeDetail{{
		StartIdx: signalIdx, StartTime: wantTime, StartClose: slots.FloatAny(chartBars[signalIdx].(map[string]any)["close"]),
	}}

	fetch := &countingBarsFetcher{}
	in := LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 3,
		ProbeRaw: probeRaw,
		Bars:     lookback, // stale: full lookback passed like an old lab snapshot
		SignalEval: eval,
		LabFetcher: WrapCachedLabBarsFetcher("sess", "run-chart-bars", fetch),
		SessionID:  "sess", RunID: "run-chart-bars",
		DetailTimeout: 0,
	}
	ref := LabDetailRef("buy", signalIdx)
	detail, err := BuildLabEpisodeDetail(context.Background(), in, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !barTimeStringsEqual(detail.Time, wantTime) {
		t.Fatalf("detail.time=%q want chart bar %q (lookback at same index was %q)", detail.Time, wantTime, wrongTime)
	}
	if detail.Close != eval.BuyDetails[0].StartClose {
		t.Fatalf("detail.close=%v want %v", detail.Close, eval.BuyDetails[0].StartClose)
	}
}

func TestEnsureProbeChartBarsForLab_ClampWhenBarsEqualsLookback(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	end := time.Date(2026, 11, 5, 10, 30, 0, 0, loc)
	var bars []any
	for d := end.AddDate(0, -5, 0); !d.After(end); d = d.Add(24 * time.Hour) {
		bars = append(bars, map[string]any{"time": d.Format("2006-01-02 15:04:05"), "close": 1.0})
	}
	probeRaw := map[string]any{"bars": append([]any(nil), bars...), "bars_lookback": append([]any(nil), bars...)}
	EnsureProbeChartBarsForLab(probeRaw, 3)
	chart, _ := probeRaw["bars"].([]any)
	lb, _ := probeRaw["bars_lookback"].([]any)
	if len(chart) >= len(lb) {
		t.Fatalf("expected clamped chart shorter than lookback got chart=%d lookback=%d", len(chart), len(lb))
	}
}
