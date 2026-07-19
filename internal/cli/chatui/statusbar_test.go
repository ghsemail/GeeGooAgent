package chatui

import (
	"strings"
	"testing"
)

func TestFormatTokenCount(t *testing.T) {
	if FormatTokenCount(19600) != "19.6K" {
		t.Fatalf("got %s", FormatTokenCount(19600))
	}
	if FormatTokenCount(1_100_000) != "1.1M" {
		t.Fatalf("got %s", FormatTokenCount(1_100_000))
	}
}

func TestRenderGrokFooterBar(t *testing.T) {
	line := stripANSI(RenderGrokFooterBar(StatusBarOptions{
		Model: "openai/gpt-5.5", PromptTokens: 19600, ContextWindow: 1_100_000,
	}, 120))
	if !strings.Contains(line, "19.6K") {
		t.Fatalf("missing tokens: %s", line)
	}
	if !strings.Contains(line, "/help") {
		t.Fatalf("missing hints: %s", line)
	}
}
