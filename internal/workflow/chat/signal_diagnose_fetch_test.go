package chat

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestApplySignalDiagnoseSeriesDataDiagnoseMerge(t *testing.T) {
	flow := &Flow{}
	res := tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"buy_hits": 2, "sell_hits": 1,
			"bars": []any{
				map[string]any{"time": time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339), "close": 1.0},
				map[string]any{"time": time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339), "close": 2.0},
			},
			"buy_merged":  []any{1, 0},
			"sell_merged": []any{0, -1},
			"verdict":     "ok",
			"summary":     "test",
		},
	}
	applySignalDiagnoseSeriesData(flow, res, 3, nil, nil, true)
	if !flow.DiagnoseMergedAtProbe {
		t.Fatal("expected merged at probe")
	}
	bars, _ := flow.ProbeRaw["bars"].([]any)
	if len(bars) == 0 {
		t.Fatal("expected probe bars after finalize")
	}
	if flow.DiagnoseRaw == nil || flow.DiagnoseVerdict != "ok" {
		t.Fatal("expected diagnose raw")
	}
}
