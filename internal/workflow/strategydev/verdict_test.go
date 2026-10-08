package strategydev

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestWorkflowSkipsSideways(t *testing.T) {
	res, err := Run(t.Context(), Input{Code: "00700.HK", Frequency: "60m"}, Deps{
		Run: func(ctx context.Context, name string, args map[string]any) tools.Result {
			if name != "probe_bot_signal_series" {
				return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{}}}
			}
			n := 40
			bars := make([]any, n)
			for i := 0; i < n; i++ {
				bars[i] = map[string]any{"close": 100.0, "high": 100.5, "low": 99.5, "volume": 10.0}
			}
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{
				"bars": bars,
				"buy_rules": []any{map[string]any{
					"index": "ADX",
					"value_series": map[string]any{
						"ADX": repeatFloat(15, n), "PLUS_DI": repeatFloat(18, n), "MINUS_DI": repeatFloat(17, n),
					},
				}},
			}}
		},
		Jev: NewJevClient(JevConfig{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Regime != RegimeRange {
		t.Fatalf("expected skip sideways, got regime=%s skipped=%v", res.Regime, res.Skipped)
	}
}
