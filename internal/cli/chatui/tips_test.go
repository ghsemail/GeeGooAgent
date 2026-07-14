package chatui

import (
	"strings"
	"testing"
)

func TestWelcomeTipLines(t *testing.T) {
	lines := welcomeTipLines()
	if len(lines) < 5 {
		t.Fatalf("expected several tips, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "SpaceX") {
		t.Fatalf("missing analysis tip: %q", lines[0])
	}
}

func TestRenderWelcomeTips(t *testing.T) {
	out := RenderWelcomeTips()
	if !strings.Contains(out, "Tips:") || !strings.Contains(out, "/help") {
		t.Fatalf("unexpected tips render: %q", out)
	}
}
