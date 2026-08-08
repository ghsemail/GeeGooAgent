package stockfmt

import (
	"strings"
	"testing"
)

func TestFormatStockNews(t *testing.T) {
	raw := `## 【个股新闻：601766.SH】

**中国中车:年度权益分派公告**
   🕐 2026-08-06
   🔗 https://example.com/a
`
	out := FormatStockNews(raw, "601766.SH")
	if strings.Contains(out, "601766") {
		t.Fatalf("should not contain code: %s", out)
	}
	if strings.Contains(out, "http") || strings.Contains(out, "🔗") {
		t.Fatalf("should not contain links: %s", out)
	}
	if !strings.Contains(out, "新闻综述") {
		t.Fatalf("missing digest: %s", out)
	}
}

func TestFormatCapitalFlowSummary(t *testing.T) {
	out := FormatCapitalFlowSummary(map[string]any{
		"main_in_flow": 1.23e8,
		"in_flow":      1.5e8,
	}, "DAY")
	if strings.Contains(out, "e") {
		t.Fatalf("scientific notation leaked: %s", out)
	}
}

func TestHumanizeScientificNumbers(t *testing.T) {
	out := HumanizeScientificNumbers("主力净流入 1.23e+08 元")
	if strings.Contains(out, "e+") {
		t.Fatalf("not humanized: %s", out)
	}
	if !strings.Contains(out, "亿") {
		t.Fatalf("expected 亿: %s", out)
	}
}
