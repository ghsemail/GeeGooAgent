package chatui

import (
	"strings"
	"testing"
	"time"
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
	line := stripANSI(RenderHermesStatusBar(StatusBarOptions{
		Model: "deepseek/deepseek-chat", PromptTokens: 19600, ContextWindow: 128_000,
		Elapsed: 12 * time.Second, Busy: true, Steps: 3,
	}, 120))
	if !strings.Contains(line, "deepseek-chat") {
		t.Fatalf("missing model: %s", line)
	}
	if !strings.Contains(line, "19.6K") {
		t.Fatalf("missing tokens: %s", line)
	}
	if !strings.Contains(line, "steps") {
		t.Fatalf("missing steps: %s", line)
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
