package chatui

import (
	"strings"
	"testing"
)

func TestRenderAssistantMarkdownGluedChineseHeadings(t *testing.T) {
	in := "##你好！我是GeeGoo的股票分析Agent，可以帮你做这些事：###1.股票分析与实时行情 我能帮你分析股票走势、技术信号、资金流向等。**实时行情-** 查询价格、K线、成交量"
	out := stripANSI(RenderAssistantMarkdown(in, 100))
	t.Logf("out=%q", out)
	if strings.Contains(out, "##") || strings.Contains(out, "###") {
		t.Fatalf("raw heading markers should be rendered away: %q", out)
	}
	if strings.Contains(out, "**") {
		t.Fatalf("raw bold markers should be rendered away: %q", out)
	}
	if !strings.Contains(out, "你好") || !strings.Contains(out, "实时行情") {
		t.Fatalf("missing content: %q", out)
	}
}

func TestPreprocessSplitsGluedH3Numbered(t *testing.T) {
	in := "前言###1.股票分析 **bold**"
	out := PreprocessTerminalMarkdown(in)
	if !strings.Contains(out, "\n###") && !strings.Contains(out, "\n##") {
		t.Fatalf("expected split before heading: %q", out)
	}
}
