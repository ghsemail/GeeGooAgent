package klinepattern

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type flakyProbeRunner struct {
	attempts atomic.Int32
}

func (s *flakyProbeRunner) Run(ctx context.Context, name string, args map[string]any) tools.Result {
	if name != "probe_bot_signal_series" {
		return tools.Result{Status: tools.StatusError}
	}
	if s.attempts.Add(1) == 1 {
		return tools.Result{Status: tools.StatusError, Summary: "market fetch failed"}
	}
	return tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": []any{map[string]any{"time": "2024-01-01", "close": 1}},
		},
	}
}

func TestFetchKlinePatternProbeSeriesWithRetry(t *testing.T) {
	testProbeHost(t)
	run := &flakyProbeRunner{}
	bars, _, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(
		context.Background(), run.Run, "00700.HK", "daily", 3, nil,
	)
	if unsup || !ok || len(bars) != 1 {
		t.Fatalf("ok=%v unsup=%v bars=%d attempts=%d", ok, unsup, len(bars), run.attempts.Load())
	}
	if run.attempts.Load() < 2 {
		t.Fatalf("attempts=%d want >=2", run.attempts.Load())
	}
}

type stubUnsupportedProbeRunner struct {
	calls []string
}

func (s *stubUnsupportedProbeRunner) Run(ctx context.Context, name string, args map[string]any) tools.Result {
	if name != "probe_bot_signal_series" {
		return tools.Result{Status: tools.StatusError}
	}
	freq, _ := args["frequency"].(string)
	s.calls = append(s.calls, freq)
	return tools.Result{Status: tools.StatusError, Summary: "unsupported frequency"}
}

func TestFetchKlinePatternProbeSeriesWithRetry_NoRetryUnsupported(t *testing.T) {
	testProbeHost(t)
	run := &stubUnsupportedProbeRunner{}
	_, _, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(
		context.Background(), run.Run, "00700.HK", "daily", 3, nil,
	)
	if ok || !unsup {
		t.Fatalf("ok=%v unsup=%v", ok, unsup)
	}
	if len(run.calls) != 1 {
		t.Fatalf("calls=%v want single attempt", run.calls)
	}
}
