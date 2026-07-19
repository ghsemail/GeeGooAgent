package chatui

import (
	"strings"
	"testing"
)

func TestRenderAssistantMarkdownHeadings(t *testing.T) {
	out := RenderAssistantMarkdown("## Title\n\n- one\n- two", 80)
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty markdown output")
	}
	if !strings.Contains(out, "Title") {
		t.Fatalf("missing heading: %q", out)
	}
}

func TestRenderAssistantBoxWithPlain(t *testing.T) {
	out := RenderAssistantBoxWith("hello **world**", 80, AssistantRenderOptions{Markdown: false})
	if !strings.Contains(out, "hello") {
		t.Fatalf("missing text: %q", out)
	}
}
