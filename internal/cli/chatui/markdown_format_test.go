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
	if !strings.Contains(out, "网格区间：315.7–506.8") {
		t.Fatalf("missing field line: %q", out)
	}
	if strings.Contains(out, "---") {
		t.Fatalf("should not keep horizontal rules: %q", out)
	}
}

func TestPreprocessTerminalMarkdown_PreservesNonTable(t *testing.T) {
	in := "结论：今日港股交易日。\n\n- 腾讯 457.6\n"
	got := PreprocessTerminalMarkdown(in)
	if !strings.Contains(got, "结论：今日港股交易日。") || !strings.Contains(got, "腾讯 457.6") {
		t.Fatalf("unexpected change: %q", got)
	}
}

func TestNormalizeAssistantLayout_GluedHeaders(t *testing.T) {
	in := "数据如下：## 🔴 腾讯控股机器人 | ID: abc | 状态: 运行中---## ⚪ 未命名"
	out := NormalizeAssistantLayout(in)
	if !strings.Contains(out, "\n## 🔴 腾讯控股机器人") {
		t.Fatalf("missing header break: %q", out)
	}
	if !strings.Contains(out, "\n## ⚪ 未命名") {
		t.Fatalf("missing second header: %q", out)
	}
	if strings.Contains(out, "---") {
		t.Fatalf("horizontal rules should be stripped: %q", out)
	}
	if !strings.Contains(out, "  ID: abc") {
		t.Fatalf("missing indented fields: %q", out)
	}
}

func TestTightenParagraphSpacing_SectionGaps(t *testing.T) {
	in := "前言\n## 标题A\n字段1\n## 标题B\n字段2"
	out := tightenParagraphSpacing(in)
	if !strings.Contains(out, "\n\n## 标题A") {
		t.Fatalf("want blank before section: %q", out)
	}
}

func TestStripInlineMarkdown(t *testing.T) {
	in := "**1. 腾讯控股** · `00700.HK`"
	got := stripInlineMarkdown(in)
	if strings.Contains(got, "*") || strings.Contains(got, "`") {
		t.Fatalf("markers should be stripped: %q", got)
	}
	if !strings.Contains(got, "腾讯控股") || !strings.Contains(got, "00700.HK") {
		t.Fatalf("content lost: %q", got)
	}
}

func TestHardWrapLine_Chinese(t *testing.T) {
	in := "这是一段很长的中文说明文字用于测试在终端里是否会强制折行显示而不是挤成一行"
	out := hardWrapLine(in, 20)
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected hard wrap newlines: %q", out)
	}
}
