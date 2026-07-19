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

func TestNormalizeAssistantLayout_GluedColonList(t *testing.T) {
	in := "可以帮你：- 行情分析：A股/港股/美股 - 交易 Bot 管理：list / create"
	out := NormalizeAssistantLayout(in)
	if !strings.Contains(out, "可以帮你：\n- 行情分析") {
		t.Fatalf("missing colon list break: %q", out)
	}
	if !strings.Contains(out, "\n- 交易 Bot") {
		t.Fatalf("missing second list item: %q", out)
	}
}

func TestRenderAssistantMarkdownSplitsGluedList(t *testing.T) {
	in := "你好！可以帮你：- 📊 行情分析：A股 - 🤖 交易 Bot"
	out := stripANSI(RenderAssistantMarkdown(in, 100))
	if !strings.Contains(out, "行情分析") {
		t.Fatalf("missing content: %q", out)
	}
	// glamour renders list items on separate visual lines
	if strings.Count(out, "\n") < 2 {
		t.Fatalf("expected multiple lines for list: %q", out)
	}
}

func TestNormalizeAssistantLayout_InlineH3Sections(t *testing.T) {
	in := "## 腾讯控股机器人 (00700.HK) — GRID 网络 Bot ### 1. 基本信息 - 代码: 00700.HK ### 2. 网格配置"
	out := NormalizeAssistantLayout(in)
	if strings.Contains(out, "###") {
		t.Fatalf("hash markers should be split/stripped: %q", out)
	}
	if !strings.Contains(out, "1. 基本信息") || !strings.Contains(out, "2. 网格配置") {
		t.Fatalf("sections missing: %q", out)
	}
}

func TestPreprocessTerminalMarkdown_GluedTableRows(t *testing.T) {
	in := `## 📌 DCA提醒机器人（6个）|#|名称|代码|频率|买入信号|状态||
||1|黄金ETF提醒器|518880.SH|60m|-|✅开启||2|中国船舶提醒机器人|600150.SH|60m|-|✅开启||3|中航沈飞提醒机器人|600760.SH|60m|-|✅开启|`
	out := PreprocessTerminalMarkdown(in)
	if strings.Contains(out, "||") {
		t.Fatalf("glued pipes should be removed: %q", out)
	}
	if !strings.Contains(out, "## 📌 DCA提醒机器人（6个）") {
		t.Fatalf("missing section title: %q", out)
	}
	if !strings.Contains(out, "**1. 黄金ETF提醒器**") {
		t.Fatalf("missing first card: %q", out)
	}
	if !strings.Contains(out, "**2. 中国船舶提醒机器人**") {
		t.Fatalf("missing second card: %q", out)
	}
	if !strings.Contains(out, "**3. 中航沈飞提醒机器人**") {
		t.Fatalf("missing third card: %q", out)
	}
	if !strings.Contains(out, "频率：60m") {
		t.Fatalf("missing field line: %q", out)
	}
}

func TestPreprocessTerminalMarkdown_GluedTableWithSectionBreak(t *testing.T) {
	in := `||1|五粮液提醒机器人|000858.SZ|120-180|10|-5%|✅开启||## 📌 Smart提醒机器人（1个）||1|腾讯控股提醒机器人|00700.HK|daily|-|✅开启|`
	out := PreprocessTerminalMarkdown(in)
	if strings.Contains(out, "||") {
		t.Fatalf("glued pipes should be removed: %q", out)
	}
	if !strings.Contains(out, "## 📌 Smart提醒机器人（1个）") {
		t.Fatalf("missing glued section header: %q", out)
	}
	if !strings.Contains(out, "**1. 五粮液提醒机器人**") {
		t.Fatalf("missing grid card: %q", out)
	}
	if !strings.Contains(out, "**1. 腾讯控股提醒机器人**") {
		t.Fatalf("missing smart card: %q", out)
	}
}

func TestExpandGluedPipeRows_SectionHeader(t *testing.T) {
	got := expandGluedPipeRows("||1|foo|bar|baz||## Title||2|qux|qaz|wsx|")
	if len(got) != 3 {
		t.Fatalf("want 3 rows, got %d: %q", len(got), got)
	}
	if got[1] != "## Title" {
		t.Fatalf("middle should be section header: %q", got[1])
	}
}

func TestHardWrapLine_Chinese(t *testing.T) {
	in := "这是一段很长的中文说明文字用于测试在终端里是否会强制折行显示而不是挤成一行"
	out := WrapPlain(in, 20)
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected hard wrap newlines: %q", out)
	}
}
