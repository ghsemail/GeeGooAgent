package memory

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestApplyKeyLevelsTool(t *testing.T) {
	ws := &StockWorkspace{Code: "00700.HK"}
	applyKeyLevelsTool(ws, map[string]any{
		"current_price": 450.0,
		"judgment": map[string]any{
			"support": map[string]any{
				"center": 440.0, "low": 438.0, "high": 442.0,
				"state": "tested", "sources": []any{"daily_val"},
			},
			"resistance": map[string]any{
				"center": 458.0, "low": 456.0, "high": 460.0,
				"state": "untested", "sources": []any{"daily_swing_high"},
			},
		},
	}, true)
	if !ws.KeyLevelsEngineOK || ws.KeyLevelSupportCenter != 440 {
		t.Fatalf("support center: %+v", ws)
	}
	if ws.KeyLevelSupportZone != "438.00~442.00" {
		t.Fatalf("zone: %s", ws.KeyLevelSupportZone)
	}
}

func TestWorkingApplyGetKeyLevels(t *testing.T) {
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-kl", "premarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.Stocks["00700.HK"] = StockWorkspace{Code: "00700.HK", Status: "pending"}
	updated, err := store.Apply(w, "get_key_levels", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"code": "00700.HK", "current_price": 450.0,
			"judgment": map[string]any{
				"support": map[string]any{"center": 442.0, "low": 440.0, "high": 444.0, "sources": []any{"poc"}},
				"resistance": map[string]any{
					"center": 457.0, "low": 455.0, "high": 459.0, "sources": []any{"swing"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks["00700.HK"]
	if !ws.KeyLevelsEngineOK {
		t.Fatal("expected engine ok")
	}
}
