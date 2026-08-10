package memory

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestApplyGetStockDailyReportsAcceptsTypedMapSlice(t *testing.T) {
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-daily", "postmarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.CurrentStock = "601766.SH"
	w.Stocks["601766.SH"] = StockWorkspace{Code: "601766.SH", Status: "pending"}
	if err := store.Save(w); err != nil {
		t.Fatal(err)
	}

	// Tool handler returns []map[string]any (DailyReportsData.PreMarket), not []any.
	updated, err := store.Apply(w, "get_stock_daily_reports", tools.Result{
		Status:  tools.StatusOK,
		Summary: "daily reports 601766.SH",
		Data: map[string]any{
			"code": "601766.SH",
			"stock_premarket": []map[string]any{{
				"result": "short", "confidence": "high", "reason": "weak open",
				"suggestion": "wait", "report_id": "rid-pre-1",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks["601766.SH"]
	if ws.PreMarketResult != "short" {
		t.Fatalf("PreMarketResult=%q want short", ws.PreMarketResult)
	}
	if ws.PreMarketConfidence != "high" || ws.PreMarketReason != "weak open" {
		t.Fatalf("confidence/reason not applied: %+v", ws)
	}
	if ws.PreMarketSuggestion != "wait" || ws.PreMarketReportID != "rid-pre-1" {
		t.Fatalf("suggestion/report_id not applied: %+v", ws)
	}
}

func TestApplyGetStockDailyReportsAcceptsAnySlice(t *testing.T) {
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-daily-any", "postmarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.CurrentStock = "00700.HK"
	w.Stocks["00700.HK"] = StockWorkspace{Code: "00700.HK", Status: "pending"}

	updated, err := store.Apply(w, "get_stock_daily_reports", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"code": "00700.HK",
			"stock_premarket": []any{
				map[string]any{"result": "long", "report_id": "rid-2"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Stocks["00700.HK"].PreMarketResult; got != "long" {
		t.Fatalf("PreMarketResult=%q want long", got)
	}
}
