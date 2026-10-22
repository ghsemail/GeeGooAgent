package llm

import "testing"

func TestNormalizeComposeMarkdownHeadingSpace(t *testing.T) {
	in := "###信号诊断评价本次策略在 probe 阶段表现良好。"
	got := NormalizeComposeMarkdown(in)
	if got == in {
		t.Fatalf("expected heading fix, got %q", got)
	}
	if got[0] != '#' {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateToLastCompleteSentence(t *testing.T) {
	in := "第一句完整。第二句未完以及严格"
	got := TruncateToLastCompleteSentence(in)
	if got != "第一句完整。" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateToLastCompleteSentenceKeepsFull(t *testing.T) {
	in := "只有一句完整。"
	if got := TruncateToLastCompleteSentence(in); got != in {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateSkipsTickerPeriodBeforeChinese(t *testing.T) {
	in := "综合评价该策略在AAPL.后续还有中文但没有完"
	got := TruncateToLastCompleteSentence(in)
	if got == "综合评价该策略在AAPL." {
		t.Fatalf("should not cut at ticker dot, got %q", got)
	}
}
