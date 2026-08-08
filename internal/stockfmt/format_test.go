package stockfmt

import (
	"strings"
	"testing"
)

func TestFormatStockNews(t *testing.T) {
	raw := `## 【个股新闻：601766.SH】

**1. 中国中车:年度权益分派公告**
**2. 中国中车:控股股东增持公告**
1. 中国中车：H股市场公告
   🕐 2026-08-06
   🔗 https://example.com/a
`
	out := FormatStockNews(raw, "601766.SH", "中国中车")
	if strings.Contains(out, "601766") {
		t.Fatalf("should not contain code: %s", out)
	}
	if strings.Contains(out, "http") || strings.Contains(out, "🔗") {
		t.Fatalf("should not contain links: %s", out)
	}
	if !strings.Contains(out, "新闻综述") {
		t.Fatalf("missing digest: %s", out)
	}
	if strings.Contains(out, "1. 中国中车") || strings.Contains(out, "2. 中国中车") {
		t.Fatalf("numbered company prefix leaked: %s", out)
	}
	if strings.Contains(out, "- 1.") || strings.Contains(out, "- 2.") {
		t.Fatalf("numbered bullets: %s", out)
	}
}

func TestPolishStockNewsSection(t *testing.T) {
	in := `**新闻综述**：1. 中国中车:公告一；2. 中国中车:公告二；3. 中国中车:公告三
- 1. 中国中车:公告一
- 2. 中国中车:公告二`
	out := PolishStockNewsSection(in, "中国中车")
	if strings.Contains(out, "1. 中国中车") {
		t.Fatalf("digest still numbered: %s", out)
	}
	if !strings.Contains(out, "共 2 条") && !strings.Contains(out, "共 3 条") {
		t.Fatalf("expected count digest: %s", out)
	}
}

func TestFormatCapitalFlowSummary(t *testing.T) {
	out := FormatCapitalFlowSummary(map[string]any{
		"main_in_flow": -1.23e7,
		"in_flow":      -1.5e7,
	}, "DAY")
	if strings.Contains(out, "e") {
		t.Fatalf("scientific notation leaked: %s", out)
	}
	if strings.Contains(out, "净流出 +") {
		t.Fatalf("double sign: %s", out)
	}
}

func TestFormatCapitalSection(t *testing.T) {
	out := FormatCapitalSection(
		-3.35e6, -8.48e6,
		3.51e6, 1.03e7, 8.29e6, 1.58e7,
		8.47e6, 8.72e6, 9.79e6, 1.95e7,
		"2026-08-07 15:28:28",
	)
	if !strings.Contains(out, "简要解读") {
		t.Fatalf("missing analysis: %s", out)
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

func TestStripEvidenceRefs(t *testing.T) {
	in := "资金面偏空[ev_abc123]；周线ev_426e8bfc5显示回落。"
	out := StripEvidenceRefs(in)
	if strings.Contains(out, "ev_") {
		t.Fatalf("ev refs remain: %s", out)
	}
}

func TestHumanizeDoesNotBreakEvidenceMarkers(t *testing.T) {
	in := "引用[ev_+426000000000000008959033344.00e8bfc5]后"
	out := HumanizeScientificNumbers(in)
	if strings.Contains(out, "亿e8") {
		t.Fatalf("corrupted marker: %s", out)
	}
}

func TestLocalizeAttitude(t *testing.T) {
	if LocalizeAttitude("neutral") != "中性" {
		t.Fatal("neutral not localized")
	}
}
