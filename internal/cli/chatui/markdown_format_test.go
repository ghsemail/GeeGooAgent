package chatui

import (
	"strings"
	"testing"
)

func TestPreprocessTerminalMarkdown_TableToCards(t *testing.T) {
	in := `## 网格交易 Bot

| # | 名称 | 代码 | 网格区间 | 档数 | 盈亏 |
|---|------|------|----------|------|------|
| 1 | 腾讯控股机器人 | 00700.HK | 315.7–506.8 | 20 | -41.26% |
| 2 | 小米集团 | 01810.HK | 10.5–18.2 | 15 | -42.10% |
`
	out := PreprocessTerminalMarkdown(in)
	if strings.Contains(out, "|---|") {
		t.Fatalf("table separator should be removed: %q", out)
	}
	if !strings.Contains(out, "**1. 腾讯控股机器人**") {
		t.Fatalf("missing card title: %q", out)
	}
	if !strings.Contains(out, "`00700.HK`") {
		t.Fatalf("missing code: %q", out)
	}
	if !strings.Contains(out, "网格区间：315.7–506.8") {
		t.Fatalf("missing field pairs: %q", out)
	}
}

func TestPreprocessTerminalMarkdown_PreservesNonTable(t *testing.T) {
	in := "结论：今日港股交易日。\n\n- 腾讯 457.6\n"
	if got := PreprocessTerminalMarkdown(in); got != in {
		t.Fatalf("unexpected change: %q", got)
	}
}

func TestNormalizeAssistantLayout_GluedHeaders(t *testing.T) {
	in := "数据如下：## 🔴 腾讯控股机器人 | ID: abc | 状态: 运行中---## ⚪ 未命名"
	out := NormalizeAssistantLayout(in)
	if !strings.Contains(out, "\n## 🔴 腾讯控股机器人") {
		t.Fatalf("missing header break: %q", out)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Fatalf("missing hr break: %q", out)
	}
	if !strings.Contains(out, "\n## ⚪ 未命名") {
		t.Fatalf("missing second header: %q", out)
	}
	if !strings.Contains(out, "- ID: abc") {
		t.Fatalf("missing pipe field bullets: %q", out)
	}
}
