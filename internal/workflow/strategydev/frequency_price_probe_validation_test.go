package strategydev

import (
	"fmt"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPriceAnalysisProbeArgsPassProbeBotSignalSeriesValidation(t *testing.T) {
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(nil), WorkspaceRoot: t.TempDir()})
	tool, ok := r.Get("probe_bot_signal_series")
	if !ok {
		t.Fatal("missing probe_bot_signal_series")
	}
	for _, freq := range []string{"5m", "60m", "daily", "weekly"} {
		args := PriceAnalysisProbeArgs("00700.HK", freq, 3)
		if err := tools.ValidateArguments(tool.Parameters, args); err != nil {
			t.Fatalf("frequency=%s validation: %v", freq, err)
		}
		buy := args["buy_signal"].([]any)[0].(map[string]any)
		if fmt.Sprint(buy["index"]) != "nosignal" || fmt.Sprint(buy["type"]) != "signal" {
			t.Fatalf("buy_signal placeholder=%v", buy)
		}
	}
}
