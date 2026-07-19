package chatui

import (
	"strings"
	"testing"
)

func TestRenderAssistantMarkdownGluedChineseHeadings(t *testing.T) {
	in := "##你好！我是GeeGoo的股票分析Agent，可以帮你做这些事：###1.股票分析与实时行情 我能帮你分析股票走势、技术信号、资金流向等。**实时行情-** 查询价格、K线、成交量"
	out := stripANSI(RenderAssistantMarkdown(in, 100))
	if !strings.Contains(out, "你好") || !strings.Contains(out, "实时行情") {
		t.Fatalf("missing content: %q", out)
	}
	// Without preprocess, glued headings may keep raw markers; glamour needs blank lines/spaces.
	if strings.Contains(out, "##") {
		t.Logf("glued heading markers remain without preprocess: %q", out)
	}
}

func TestPreprocessSplitsGluedH3Numbered(t *testing.T) {
	in := "前言###1.股票分析 **bold**"
	out := PreprocessTerminalMarkdown(in)
	if !strings.Contains(out, "\n###") && !strings.Contains(out, "\n##") {
		t.Fatalf("expected split before heading: %q", out)
	}
}
