package klinepattern

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestEnsurePatternHitsOnDailyBarsAt_NoNilAsOfRetryForMacro(t *testing.T) {
	testProbeHost(t)
	var withAsOf, withNil int
	run := func(ctx context.Context, name string, args map[string]any) tools.Result {
		_ = ctx
		if name == "probe_bot_signal_series" {
			if _, has := args["as_of"]; has {
				withAsOf++
			} else {
				withNil++
			}
			return tools.Result{Status: tools.StatusError}
		}
		return tools.Result{Status: tools.StatusError}
	}
	asOf := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	bars := []any{map[string]any{"time": "2026-11-16 00:00:00", "close": 1.0}}
	_ = ensurePatternHitsOnDailyBarsAt(context.Background(), run, "000300.SH", 3, bars, nil, &asOf, false)
	if withNil != 0 {
		t.Fatalf("nil as_of probes=%d want 0", withNil)
	}
	if withAsOf == 0 {
		t.Fatal("expected session as_of probe attempts")
	}
}