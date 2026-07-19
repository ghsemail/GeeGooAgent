package chatui

import (
	"strings"
	"testing"
	"time"
)

func TestRenderUserPromptBoxPrefix(t *testing.T) {
	out := stripANSI(RenderUserPromptBox("你好", 60))
	if !strings.Contains(out, "> 你好") {
		t.Fatalf("out=%q", out)
	}
}

func TestRenderTurnFooter(t *testing.T) {
	out := stripANSI(RenderTurnFooter(8200 * time.Millisecond))
	if out != "Worked for 8.2s." {
		t.Fatalf("out=%q", out)
	}
}

func TestAnchorContentBottomPadsShortTranscript(t *testing.T) {
	content := "line1\nline2\nline3"
	out := AnchorContentBottom(content, 8)
	if strings.Count(out, "\n") != 7 {
		t.Fatalf("lines=%d", strings.Count(out, "\n"))
	}
	if !strings.HasSuffix(out, "line3") {
		t.Fatalf("content should end at bottom: %q", out)
	}
}

func TestAnchorContentBottomSkipsLongTranscript(t *testing.T) {
	content := strings.Repeat("x\n", 20)
	out := AnchorContentBottom(content, 8)
	if out != content {
		t.Fatalf("should not pad long content")
	}
}

func TestContentWrapWidthUsesFullTerminal(t *testing.T) {
	if got := ContentWrapWidth(200); got != 196 {
		t.Fatalf("got %d want 196", got)
	}
}
