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

func TestRenderHermesStatusBar(t *testing.T) {
	line := RenderHermesStatusBar(StatusBarOptions{
		Model: "openai/gpt-5.5", PromptTokens: 19600, ContextWindow: 1_100_000,
		Busy: true,
	}, 120)
	if !strings.Contains(line, "gpt-5.5") {
		t.Fatalf("missing model: %s", line)
	}
	if !strings.Contains(line, "19.6K") {
		t.Fatalf("missing tokens: %s", line)
	}
}
