package chatui

import (
	"strings"
	"testing"
)

func TestRenderGrokProcessHeader(t *testing.T) {
	out := stripANSI(RenderGrokProcessHeader(false, "💭 思考", 6, 1.2))
	if !strings.Contains(out, "▸") || !strings.Contains(out, "💭 思考") || !strings.Contains(out, "6 行") {
		t.Fatalf("header=%q", out)
	}
}

func TestRenderGrokReplyBlockAddsParagraphGap(t *testing.T) {
	in := "第一段。\n第二段。"
	out := stripANSI(RenderGrokReplyBlock(in, 80))
	if !strings.Contains(out, "\n\n") {
		t.Fatalf("expected paragraph gap: %q", out)
	}
}
