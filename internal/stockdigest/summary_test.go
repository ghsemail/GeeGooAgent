package stockdigest_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildFeishuSummaryPremarketStock(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "premarket_stock")
	w.Market = "HK"
	w.ReportDate = "2026-08-18"
	w.IsTradingDay = &trading
	w.MarketReportSummary = "不应出现在个股推送里"
	w.Stocks = map[string]memory.StockWorkspace{
		"00700.HK": {
			Code: "00700.HK", StockName: "腾讯控股", Status: "reported",
			Attitude: "bearish",
			PreMarketResult: "short", PreMarketConfidence: "medium", PreMarketSuggestion: "sell",
			PreMarketReason: "Bot 昨日态度为 偏空；周线技术分析已纳入；主力资金证据已纳入；共 5 条证据引用。",
			ReportSummary:      "市场背景 恒指近 5 日累计下跌约 1.38% 后于 25,453 附近企稳…个股新闻 **新闻综述**：共 8 条相关资讯",
			CapitalFlowSummary: "**资金概况**：主力净流出。\n**简要解读**：主力净流出，散户承接。",
		},
	}
	text := stockdigest.Build("premarket_stock", "HK", workflow.RunResult{Working: w, Status: "completed"})
	if !strings.Contains(text, "## 盘前个股 · HK · 2026-08-18") {
		t.Fatalf("missing title: %s", text)
	}
	if strings.Contains(text, "不应出现在个股推送里") {
		t.Fatalf("market summary should not appear: %s", text)
	}
	if strings.Contains(text, "证据已纳入") {
		t.Fatalf("boilerplate reason should be replaced: %s", text)
	}
	if !strings.Contains(text, "### 腾讯控股") || !strings.Contains(text, "#### 操作建议") {
		t.Fatalf("missing suggestion block: %s", text)
	}
	if strings.Contains(text, "市场背景") || strings.Contains(text, "新闻综述") {
		t.Fatalf("report excerpt should not appear in digest: %s", text)
	}
	if !strings.Contains(text, "腾讯控股盘前研判看空") {
		t.Fatalf("missing one-liner summary: %s", text)
	}
}

func TestBuildFeishuSummaryPostmarketStock(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Market = "HK"
	w.ReportDate = "2026-08-18"
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"00700.HK": {
			Code: "00700.HK", StockName: "腾讯控股", Status: "reported",
			ChangePct: -0.9, SessionBias: "neutral", VsPreMarket: "partial",
			ReportSummary:           "腾讯收跌，科技板块带动",
			ReportMarketSummary:     "全天震荡上行",
			ReportTradeSummary:      "网格无成交",
			ReportExperienceSummary: "维持观望等待回踩",
		},
	}
	text := stockdigest.Build("postmarket_stock", "HK", workflow.RunResult{Working: w, Status: "completed"})
	if !strings.Contains(text, "## 盘后个股 · HK") {
		t.Fatalf("missing title: %s", text)
	}
	if !strings.Contains(text, "-0.90%") || !strings.Contains(text, "部分一致") {
		t.Fatalf("missing localized metrics: %s", text)
	}
	if strings.Contains(text, "partial") {
		t.Fatalf("raw enum should be localized: %s", text)
	}
}

func TestBuildFeishuSummaryPostmarketStockSkippedWithExistingReport(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Market = "CN"
	w.ReportDate = "2026-08-20"
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"601766.SH": {
			Code: "601766.SH", StockName: "中国中车", Status: "skipped",
			ReportID: "rid-post-1", ChangePct: 0.83, SessionBias: "bullish", VsPreMarket: "partial",
			ReportSummary:           "中国中车今日收涨0.83%",
			ReportMarketSummary:     "震荡上行",
			ReportTradeSummary:      "无成交",
			ReportExperienceSummary: "维持观望",
		},
	}
	text := stockdigest.Build("postmarket_stock", "CN", workflow.RunResult{Working: w, Status: "completed"})
	if strings.Contains(text, "今日无个股盘后报告") {
		t.Fatalf("skipped existing report should appear in digest: %s", text)
	}
	if !strings.Contains(text, "### 中国中车") || !strings.Contains(text, "+0.83%") {
		t.Fatalf("missing postmarket digest body: %s", text)
	}
}
