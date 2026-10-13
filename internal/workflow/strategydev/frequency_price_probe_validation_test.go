package strategydev

import (
	"fmt"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPriceAnalysisProbeArgsPassProbeBotSignalSeriesValidation(t *testing.T) {
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(nil), WorkspaceRoot: t.TempDir()})
	tool, ok := r.Get("probe_bot_signal_series")
	if !ok {
		t.Fatal("missing probe_bot_signal_series")
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	asOf := time.Date(2026, 10, 13, 16, 0, 0, 0, loc)
	for _, freq := range []string{"5m", "60m", "daily", "weekly"} {
		args := PriceAnalysisProbeArgs("00700.HK", freq, 3)
		if err := tools.ValidateArguments(tool.Parameters, args); err != nil {
			t.Fatalf("frequency=%s validation: %v", freq, err)
		}
		withAsOf := PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{Code: "00700.HK", Freq: freq, Months: 3, AsOf: &asOf})
		if err := tools.ValidateArguments(tool.Parameters, withAsOf); err != nil {
			t.Fatalf("frequency=%s with as_of validation: %v", freq, err)
		}
		buy := args["buy_signal"].([]any)[0].(map[string]any)
		if fmt.Sprint(buy["index"]) != "nosignal" || fmt.Sprint(buy["type"]) != "signal" {
			t.Fatalf("buy_signal placeholder=%v", buy)
		}
	}
}
