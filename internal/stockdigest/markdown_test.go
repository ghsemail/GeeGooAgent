package stockdigest_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
)

func TestFormatReasonMarkdownEvidenceBullets(t *testing.T) {
	t.Parallel()
	reason := "多重证据共振偏空：【】周线级别累计跌 -12.31%【】当日主力资金净流出 5.52 亿【】Bot 昨日态度已为 bearish"
	out := stockdigest.FormatReasonMarkdownForTest(reason)
	if !strings.Contains(out, "**多重证据共振偏空**") {
		t.Fatalf("missing prefix: %s", out)
	}
	if strings.Count(out, "\n- ") < 2 {
		t.Fatalf("expected bullet list: %s", out)
	}
	if strings.Contains(out, "【】") {
		t.Fatalf("raw markers should be removed: %s", out)
	}
}

func TestFeishuDigestPremarketMarkdown(t *testing.T) {
	t.Parallel()
	text := stockdigest.FormatPremarketStockForTest(memory.StockWorkspace{
		Code: "00700.HK", StockName: "腾讯控股",
		Attitude: "bearish",
		PreMarketResult: "short", PreMarketConfidence: "medium", PreMarketSuggestion: "sell",
		PreMarketReason: "多重证据共振偏空：【】周线级别累计跌 -12.31%【】当日主力资金净流出 5.52 亿",
		ReportSummary:   "腾讯周线跌破所有均线，建议顺势偏空操作。",
	})
	for _, want := range []string{"### 腾讯控股（00700.HK）", "- **结论**：看空", "#### 操作建议", "#### 摘要", "#### 判定依据", "> 可考虑减仓"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}
