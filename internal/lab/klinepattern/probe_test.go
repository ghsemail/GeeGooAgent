package klinepattern

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type stubKlineProbeRunner struct {
	calls  []string
	fail4h bool
}

func (s *stubKlineProbeRunner) Run(ctx context.Context, name string, args map[string]any) tools.Result {
	if name != "probe_bot_signal_series" {
		return tools.Result{Status: tools.StatusError}
	}
	freq, _ := args["frequency"].(string)
	s.calls = append(s.calls, freq)
	if freq == "240m" && s.fail4h {
		return tools.Result{Status: tools.StatusError, Summary: "unsupported frequency"}
	}
	return tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": []any{map[string]any{"time": "2024-01-01", "close": 1}},
		},
	}
}

func testProbeHost(t *testing.T) {
	t.Helper()
	SetHost(Host{
		ProbeArgsExtended: func(code, freq string, months int, asOf *time.Time) map[string]any {
			args := map[string]any{
				"code": code, "frequency": freq, "months_back": months,
			}
			if asOf != nil {
				args["as_of"] = asOf.Format(time.RFC3339)
			}
			return args
		},
		MonthsBackForPerSignalFetch: func(chartMonths int, chartBars []any, freq string, windowEnd time.Time) int {
			return CapMonthsBack(chartMonths)
		},
	})
}

func TestBuildKlinePatternMTFBundle_SixtyMinuteIncludesFourHour(t *testing.T) {
	testProbeHost(t)
	runOK := &stubKlineProbeRunner{}
	bundle := BuildKlinePatternMTFBundle(context.Background(), runOK.Run, "00700.HK", "60m", 3, nil)
	if bundle == nil {
		t.Fatal("nil bundle")
	}
	if !probeCallsContain(runOK.calls, "240m") {
		t.Fatalf("60m ladder must probe 240m for 4h slot, calls=%v", runOK.calls)
	}
	for _, want := range []string{"60m", "4h", "daily", "weekly"} {
		if _, ok := bundle.ByFreq[want]; !ok {
			t.Fatalf("missing ByFreq[%s] keys=%v", want, bundle.ByFreq)
		}
	}
}

func probeCallsContain(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

