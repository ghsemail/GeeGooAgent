package strategydev

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// slowMTFProbeRunner sleeps per probe call to simulate cold probe_bot_signal_series I/O.
type slowMTFProbeRunner struct {
	delay time.Duration
	mu    sync.Mutex
	calls []string
}

func (s *slowMTFProbeRunner) Run(ctx context.Context, name string, args map[string]any) tools.Result {
	if name != "probe_bot_signal_series" {
		return tools.Result{Status: tools.StatusError}
	}
	freq, _ := args["frequency"].(string)
	s.mu.Lock()
	s.calls = append(s.calls, freq)
	s.mu.Unlock()
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return tools.Result{Status: tools.StatusError, Summary: "cancelled"}
	}
	if ctx.Err() != nil {
		return tools.Result{Status: tools.StatusError, Summary: "cancelled"}
	}
	return tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": []any{map[string]any{
				"time": "2026-06-01 10:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5, "volume": 100.0,
			}},
		},
	}
}

func TestBuildLabEpisodeDetail_MTFDailyWeeklyFailUnder8sBudget(t *testing.T) {
	// Parallel MTF probes take max(individual latency); >8s per call exceeds legacy budget.
	const perProbe = 9 * time.Second
	run := &slowMTFProbeRunner{delay: perProbe}
	bars := []any{map[string]any{"time": "2026-06-01 10:00:00", "close": 300.0}}
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Run: ToolRunner(run.Run), Code: "SPCX.US", SignalFreq: "60m", Months: 3, Bars: bars,
		LabFetcher: stubLabBarsOK{}, DetailTimeout: 8 * time.Second, PATimeout: 45 * time.Second,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.KlinePattern == nil || detail.KlinePattern.MTF == nil {
		t.Fatal("expected kline MTF block")
	}
	var dailyFail, weeklyFail bool
	for _, tf := range detail.KlinePattern.MTF.Timeframes {
		if tf.Frequency != "daily" && tf.Frequency != "weekly" {
			continue
		}
		if strings.Contains(tf.AlignNote, "拉数失败") {
			switch tf.Frequency {
			case "daily":
				dailyFail = true
			case "weekly":
				weeklyFail = true
			}
		}
	}
	if !dailyFail || !weeklyFail {
		t.Fatalf("expected daily+weekly 拉数失败 under 8s ctx with %v/probe; mtf=%+v", perProbe, detail.KlinePattern.MTF.Timeframes)
	}
}

func TestBuildLabEpisodeDetail_MTFDailyWeeklyOKWithExtendedBudget(t *testing.T) {
	const perProbe = 9 * time.Second
	run := &slowMTFProbeRunner{delay: perProbe}
	bars := []any{map[string]any{"time": "2026-06-01 10:00:00", "close": 300.0}}
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Run: ToolRunner(run.Run), Code: "SPCX.US", SignalFreq: "60m", Months: 3, Bars: bars,
		LabFetcher: stubLabBarsOK{}, DetailTimeout: DefaultLabDetailPhaseTimeout, PATimeout: 45 * time.Second,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.KlinePattern == nil || detail.KlinePattern.MTF == nil {
		t.Fatal("expected kline MTF block")
	}
	for _, tf := range detail.KlinePattern.MTF.Timeframes {
		if tf.Frequency != "daily" && tf.Frequency != "weekly" {
			continue
		}
		if strings.Contains(tf.AlignNote, "拉数失败") {
			t.Fatalf("%s should not 拉数失败 with %v budget: %+v", tf.Frequency, DefaultLabDetailPhaseTimeout, tf)
		}
	}
}
