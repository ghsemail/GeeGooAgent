package stockfmt_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func TestFormatIntradayHourlySectionStripsTables(t *testing.T) {
	raw := `### 一、整体走势概览

| 日期 | 收盘 | 涨跌 |
| --- | --- | --- |
| 7/30 | 5.88 | +1.2% |

- 价格围绕 5.88 元震荡

### 三、综合结论

当前偏多，关注量能配合。`
	out := stockfmt.FormatIntradayHourlySection(raw, "fallback")
	if strings.Contains(out, "|") {
		t.Fatalf("expected no table pipes: %s", out)
	}
	if strings.Contains(out, "###") {
		t.Fatalf("expected no h3 headings: %s", out)
	}
	if !strings.Contains(out, "5.88") {
		t.Fatalf("expected price retained: %s", out)
	}
}

func TestFormatIntradaySignalSectionKeepsVerdict(t *testing.T) {
	raw := `### 二、各技术指标最新状态

| EMA | MACD | KDJ |
| --- | --- | --- |
| 多头 | 金叉 | 超买 |

### 三、综合多空判定

信号偏多，建议顺势参与。`
	out := stockfmt.FormatIntradaySignalSection(raw)
	if strings.Contains(out, "EMA") && strings.Contains(out, "|") {
		t.Fatalf("expected metric table removed: %s", out)
	}
	if !strings.Contains(out, "偏多") {
		t.Fatalf("expected verdict retained: %s", out)
	}
}

func TestLocalizeDecisionTerms(t *testing.T) {
	in := "决策 buy，置信度 high，盘前 long。"
	out := stockfmt.LocalizeDecisionTerms(in)
	if strings.Contains(strings.ToLower(out), "buy") || strings.Contains(out, "high") {
		t.Fatalf("expected English enums localized: %s", out)
	}
	if !strings.Contains(out, "买入") || !strings.Contains(out, "高") || !strings.Contains(out, "看多") {
		t.Fatalf("expected Chinese terms: %s", out)
	}
}

func TestIntradayAPISummaryNoTags(t *testing.T) {
	s := stockfmt.IntradayAPISummary("中国中车", "601766.SH", "信号买入", "buy", "high", "## 小时级分析\n\n偏多。")
	if strings.Contains(s, "intraday") || strings.Contains(s, "buy") {
		t.Fatalf("unexpected tag or English enum: %s", s)
	}
	if !strings.Contains(s, "买入") || !strings.Contains(s, "高") {
		t.Fatalf("expected localized summary: %s", s)
	}
}
