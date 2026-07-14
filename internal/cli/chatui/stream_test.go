package chatui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func TestTypewriterStreamFinalReply(t *testing.T) {
	t.Setenv("GEEGOO_CHAT_PLAIN", "1")
	var buf bytes.Buffer
	ui := chatui.New(&buf)

	ui.EmitProgress("turn_start", nil)
	ui.EmitProgress("stream_delta", map[string]any{"content": "你好"})
	ui.EmitProgress("stream_delta", map[string]any{"content": "世界"})
	if !ui.FinishAssistantStream() {
		t.Fatal("expected streamed final reply")
	}
	out := buf.String()
	if !strings.Contains(out, "你好世界") {
		t.Fatalf("missing streamed text: %q", out)
	}
	if !strings.Contains(out, "GeeGoo>") {
		t.Fatalf("missing stream header: %q", out)
	}
}

func TestTypewriterAbortedBeforeTools(t *testing.T) {
	t.Setenv("GEEGOO_CHAT_PLAIN", "1")
	var buf bytes.Buffer
	ui := chatui.New(&buf)

	ui.EmitProgress("turn_start", nil)
	ui.EmitProgress("stream_delta", map[string]any{"content": "先查一下"})
	ui.EmitProgress("llm_tools", map[string]any{"tool_names": []string{"search_code"}})
	ui.EmitProgress("stream_delta", map[string]any{"content": "腾讯是 00700"})
	if !ui.FinishAssistantStream() {
		t.Fatal("expected final streamed reply after tools")
	}
	out := buf.String()
	if !strings.Contains(out, "腾讯是 00700") {
		t.Fatalf("missing final reply: %q", out)
	}
	// Plan text was started then aborted; final answer must still stream.
	if strings.Count(out, "GeeGoo>") < 2 {
		t.Fatalf("expected plan+final stream headers, got: %q", out)
	}
}

func TestTypewriterSkipsDuplicatePlanWhenStreamed(t *testing.T) {
	t.Setenv("GEEGOO_CHAT_PLAIN", "1")
	var buf bytes.Buffer
	ui := chatui.New(&buf)

	ui.EmitProgress("turn_start", nil)
	ui.EmitProgress("stream_delta", map[string]any{"content": "计划A"})
	ui.EmitProgress("llm_plan", map[string]any{
		"content": "计划A", "tool_names": []string{"search_code"},
	})
	out := buf.String()
	if strings.Contains(out, "[计划]") {
		t.Fatalf("llm_plan should not reprint streamed content: %q", out)
	}
}
