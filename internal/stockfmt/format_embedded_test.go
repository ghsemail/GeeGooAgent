package stockfmt_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func TestFormatEmbeddedHourlyAnalysisDemotesHeadings(t *testing.T) {
	raw := `### 一、整体走势概览

- 价格 5.88

### 二、阶段分析

- 日期：成交量（手）：备注
- 7/30：100：峰值

> **结论：** 当前中国中车（601766.`
	out := stockfmt.FormatEmbeddedHourlyAnalysis(raw)
	if strings.Contains(out, "###") {
		t.Fatalf("expected no h3 headings: %s", out)
	}
	if !strings.Contains(out, "• **一、整体走势概览**") {
		t.Fatalf("expected cn section bullet: %s", out)
	}
	if strings.Contains(out, "日期：成交量") {
		t.Fatalf("expected garbage table line removed: %s", out)
	}
	if strings.Contains(out, "601766.") {
		t.Fatalf("expected truncated conclusion removed: %s", out)
	}
}
