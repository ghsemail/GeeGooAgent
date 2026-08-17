package scheduler_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildFeishuSummaryPremarketStock(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "premarket_stock")
	w.Market = "CN"
	w.ReportDate = "2026-08-17"
	w.IsTradingDay = &trading
	w.MarketReportSummary = "A股高开震荡，科技领涨"
	w.Stocks = map[string]memory.StockWorkspace{
		"00700.HK": {
			Code: "00700.HK", StockName: "腾讯控股", Status: "reported",
			PreMarketResult: "long", PreMarketConfidence: "high", PreMarketSuggestion: "buy",
			PreMarketReason: "小时线多头排列", ReportID: "abc123",
		},
	}
	job := scheduler.Job{Name: "premarket_stock_cn", Skill: "premarket_stock", Market: "CN", Platform: "feishu"}
	text := scheduler.BuildFeishuSummary(job, workflow.RunResult{Working: w, Status: "completed"})
	if !strings.Contains(text, "【盘前准备·CN】") {
		t.Fatalf("missing title: %s", text)
	}
	if !strings.Contains(text, "腾讯控股") || !strings.Contains(text, "abc123") {
		t.Fatalf("missing stock: %s", text)
	}
}

func TestBuildFeishuSummaryPostmarketStock(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Market = "HK"
	w.ReportDate = "2026-08-17"
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"00700.HK": {
			Code: "00700.HK", StockName: "腾讯控股", Status: "reported",
			ChangePct: 1.5, SessionBias: "bullish", VsPreMarket: "aligned",
			BotLogSummary: "网格无成交", ReportID: "post-1",
			HourlyPriceAnalysis: "收盘走强",
		},
	}
	job := scheduler.Job{Name: "postmarket_stock_hk", Skill: "postmarket_stock", Market: "HK", Platform: "feishu"}
	text := scheduler.BuildFeishuSummary(job, workflow.RunResult{Working: w, Status: "completed"})
	if !strings.Contains(text, "【盘后总结·HK】") {
		t.Fatalf("missing title: %s", text)
	}
	if !strings.Contains(text, "+1.50%") || !strings.Contains(text, "aligned") {
		t.Fatalf("missing metrics: %s", text)
	}
}

func TestShouldNotifyJob(t *testing.T) {
	t.Parallel()
	if !scheduler.ShouldNotifyJobForTest(scheduler.Job{Skill: "premarket_stock", Platform: "feishu"}) {
		t.Fatal("expected notify")
	}
	if scheduler.ShouldNotifyJobForTest(scheduler.Job{Skill: "premarket_market", Platform: "feishu"}) {
		t.Fatal("market job should not notify")
	}
}
