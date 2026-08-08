package memory_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestWorkingStoreApplyHourlyBundle(t *testing.T) {
	store := memory.NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-bundle", "postmarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.Stocks["00700.HK"] = memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股", Status: "collecting"}
	if err := store.Save(w); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Apply(w, "get_hourly_analysis_bundle", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"code": "00700.HK",
			"price_analysis":  "price trend up",
			"signal_analysis": "macd bullish",
			"kline_analysis":  "hourly kline strong",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks["00700.HK"]
	if ws.HourlyPriceAnalysis != "price trend up" || ws.HourlySignalAnalysis != "macd bullish" || ws.HourlyKlineAnalysis != "hourly kline strong" {
		t.Fatalf("hourly fields not applied: %+v", ws)
	}
	if len(updated.EvidenceRefs) != 3 {
		t.Fatalf("expected 3 evidence refs, got %d", len(updated.EvidenceRefs))
	}
}
