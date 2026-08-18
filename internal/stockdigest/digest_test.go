package stockdigest_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
)

func TestFeishuDigestPremarketFormatting(t *testing.T) {
	t.Parallel()
	text := stockdigest.FormatPremarketStockForTest(memory.StockWorkspace{
		Code: "00700.HK", StockName: "腾讯控股",
		Attitude: "bearish",
		PreMarketResult: "short", PreMarketConfidence: "medium", PreMarketSuggestion: "sell",
		PreMarketReason: "Bot 昨日态度为 偏空；周线技术分析已纳入；主力资金证据已纳入；共 5 条证据引用。",
		ReportSummary:      "市场背景 恒指…个股新闻 **新闻综述**：共 8 条",
		CapitalFlowSummary: "**简要解读**：主力净流出，散户承接。",
	})
	if !strings.HasPrefix(text, "### 腾讯控股（00700.HK）") {
		t.Fatalf("unexpected header: %s", text)
	}
	if !strings.Contains(text, "- **结论**：看空") {
		t.Fatalf("missing decision bullet: %s", text)
	}
}
