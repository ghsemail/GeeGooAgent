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

func TestRenderAssistantBoxWithLiveUsesPlain(t *testing.T) {
	out := stripANSI(RenderAssistantBoxWith("## Title", 80, AssistantRenderOptions{Markdown: true, Live: true}))
	if strings.Contains(out, "##") && !strings.Contains(out, "Title") {
		t.Fatalf("live preview should stay plain: %q", out)
	}
}

func TestRenderAssistantBoxCompletedUsesMarkdown(t *testing.T) {
	out := stripANSI(RenderAssistantBoxWith("## Title\n\n- one\n- two", 80, AssistantRenderOptions{Markdown: true, Live: false}))
	if strings.Contains(out, "##") {
		t.Fatalf("completed reply should render markdown: %q", out)
	}
	if !strings.Contains(out, "Title") || !strings.Contains(out, "one") {
		t.Fatalf("missing content: %q", out)
	}
}

func TestRenderAssistantBoxHermesPanel(t *testing.T) {
	out := RenderAssistantBoxWith("核心能力:\n\n文件 & 代码\n- 读写文件\n- grep 搜索", 100, AssistantRenderOptions{Markdown: true})
	plain := stripANSI(out)
	if !strings.Contains(plain, "读写文件") {
		t.Fatalf("missing list item: %q", plain)
	}
	if !strings.Contains(out, "│") {
		t.Fatalf("expected Hermes panel border: %q", plain)
	}
}

func TestRenderAssistantBoxLiveNoPanel(t *testing.T) {
	out := RenderAssistantBoxWith("## Title", 80, AssistantRenderOptions{Markdown: true, Live: true})
	if strings.Contains(out, "│") {
		t.Fatalf("live preview should not use panel: %q", stripANSI(out))
	}
}
