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

func TestStripBrokenBoldMarkers(t *testing.T) {
	in := "周表现：**-1.40%** (从490.40降至478.80) 周振幅：**4.51%**"
	out := stockfmt.StripBrokenBoldMarkers(in)
	if strings.Contains(out, "%**") {
		t.Fatalf("expected broken percent markers removed: %s", out)
	}
	if !strings.Contains(out, "**整体走势**") {
		withTitle := stockfmt.RepairMCPBoldArtifacts("**整体走势** -1.40%")
		if !strings.Contains(withTitle, "**整体走势**") {
			t.Fatalf("expected section titles to stay bold: %s", withTitle)
		}
	}
	if !strings.Contains(out, "-1.40%") || !strings.Contains(out, "4.51%") {
		t.Fatalf("expected metrics retained: %s", out)
	}
}

func TestFormatIntradayHourlySectionDropsRawDailyRows(t *testing.T) {
	raw := `一、整体走势概览

统计区间：2026年7月29日 - 8月7日

二、日线级别分析

07-29：6.15：6.17：6.20：6.13：+0.33%：101.05
07-30：6.17：6.33：6.35：6.16：+2.59%：290.80

三、关键走势特征

1. 冲高回落后进入缩量阴跌，低点逐步下移。`
	out := stockfmt.FormatIntradayHourlySection(raw, "fallback")
	if strings.Contains(out, "07-29：6.15") {
		t.Fatalf("expected raw daily rows removed: %s", out)
	}
	if !strings.Contains(out, "关键走势特征") || !strings.Contains(out, "缩量阴跌") {
		t.Fatalf("expected narrative retained: %s", out)
	}
	if !strings.Contains(out, "**") {
		t.Fatalf("expected bold section titles: %s", out)
	}
}

func TestFormatIntradaySignalSectionPolishesVerdict(t *testing.T) {
	raw := `四、综合判定结论：

> ### ⚠️ 短期：偏看空（短线存在回调压力） > ### 中长期：中性略偏多（上行趋势尚未破坏）`
	out := stockfmt.FormatIntradaySignalSection(raw)
	if strings.Contains(out, "###") || strings.Contains(out, ">") {
		t.Fatalf("expected cleaned verdict: %s", out)
	}
	if !strings.Contains(out, "**短期**") || !strings.Contains(out, "**中长期**") {
		t.Fatalf("expected bold verdict labels: %s", out)
	}
}

func TestFormatPriceTrimsZeros(t *testing.T) {
	if got := stockfmt.FormatPrice(478.8); got != "478.8" {
		t.Fatalf("FormatPrice(478.8)=%q want 478.8", got)
	}
	if got := stockfmt.FormatPrice(100); got != "100" {
		t.Fatalf("FormatPrice(100)=%q want 100", got)
	}
}

func TestRepairIntradayLineBreaksJoinsMidSentenceDates(t *testing.T) {
	in := "震荡上行并在\n\n8月5日触及497.8的周期高点，随后连续两日回调。"
	out := stockfmt.RepairIntradayLineBreaks(in)
	if strings.Contains(out, "并在\n") || strings.Contains(out, "并在\n\n8月") {
		t.Fatalf("expected joined date line, got %q", out)
	}
	if !strings.Contains(out, "并在8月5日触及") {
		t.Fatalf("expected continuous sentence, got %q", out)
	}
}

func TestRepairIntradayLineBreaksKeepsSectionBreaks(t *testing.T) {
	in := "趋势偏弱。\n\n**量价关系**\n\n量能萎缩。"
	out := stockfmt.RepairIntradayLineBreaks(in)
	if !strings.Contains(out, "**量价关系**") {
		t.Fatalf("expected section header retained: %s", out)
	}
}

func TestFormatIntradayHourlySectionKeepsTrendSection(t *testing.T) {
	raw := `### 一、数据区间

数据区间：2026年8月3日—8月7日

### 二、周表现

**周表现：-1.40%** (从490.40降至478.80)

### 三、走势分析

价格围绕 478 元震荡，短线偏弱但未破位。

### 四、综合结论

观望为主。`
	out := stockfmt.FormatIntradayHourlySection(raw, "fallback")
	if strings.Contains(out, "%**") {
		t.Fatalf("expected no broken percent markers: %s", out)
	}
	if !strings.Contains(out, "走势分析") || !strings.Contains(out, "478") {
		t.Fatalf("expected trend section retained: %s", out)
	}
	if strings.Contains(out, "|") {
		t.Fatalf("expected no tables: %s", out)
	}
}
