package memory

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestApplyListTodayPostmarketReportsHydratesSkippedStock(t *testing.T) {
	t.Parallel()
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-skip-post", "postmarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.CurrentStock = "601766.SH"
	w.Stocks["601766.SH"] = StockWorkspace{Code: "601766.SH", StockName: "中国中车", Status: "pending"}

	updated, err := store.Apply(w, "list_today_stock_postmarket_reports", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"code":             "601766.SH",
			"already_reported": true,
			"reports": []map[string]any{{
				"report_id":           "rid-post-1",
				"code":                "601766.SH",
				"stock_name":          "中国中车",
				"summary":             "中国中车今日收涨0.83%",
				"market_summary":      "震荡上行",
				"trade_summary":       "无成交",
				"experience_summary":  "维持观望",
				"session_bias":        "bullish",
				"vs_stock_premarket":  "partial",
				"change_pct":          0.83,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks["601766.SH"]
	if ws.Status != "skipped" {
		t.Fatalf("Status=%q want skipped", ws.Status)
	}
	if ws.ReportID != "rid-post-1" {
		t.Fatalf("ReportID=%q", ws.ReportID)
	}
	if ws.ReportSummary != "中国中车今日收涨0.83%" {
		t.Fatalf("ReportSummary=%q", ws.ReportSummary)
	}
	if ws.ChangePct != 0.83 || ws.SessionBias != "bullish" {
		t.Fatalf("ChangePct/SessionBias not applied: %+v", ws)
	}
}

func TestApplyListTodayReportsHydratesSkippedPremarket(t *testing.T) {
	t.Parallel()
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-skip-pre", "premarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.CurrentStock = "00700.HK"
	w.Stocks["00700.HK"] = StockWorkspace{Code: "00700.HK", Status: "pending"}

	updated, err := store.Apply(w, "list_today_reports", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"code":             "00700.HK",
			"already_reported": true,
			"reports": []map[string]any{{
				"report_id":  "rid-pre-1",
				"result":     "short",
				"confidence": "medium",
				"reason":     "weak tape",
				"suggestion": "sell",
				"summary":    "腾讯盘前偏空",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks["00700.HK"]
	if ws.Status != "skipped" {
		t.Fatalf("Status=%q want skipped", ws.Status)
	}
	if ws.PreMarketResult != "short" || ws.ReportSummary != "腾讯盘前偏空" {
		t.Fatalf("premarket fields not applied: %+v", ws)
	}
}
