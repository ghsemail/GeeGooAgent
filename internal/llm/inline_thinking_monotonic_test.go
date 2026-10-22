package llm

import (
	"strings"
	"testing"
)

func TestAnswerStreamAccumulatorIgnoresVisibleShrinkGlitch(t *testing.T) {
	var visible string
	acc := AnswerStreamAccumulator{
		OnVisible: func(chunk string) { visible += chunk },
	}
	acc.PushContent("综合统计评价")
	acc.PushContent("<think>x</think>")
	if visible == "" || !strings.HasPrefix(visible, "综合") {
		t.Fatalf("visible=%q", visible)
	}
}
