package agent

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestStreamHandlerToolGenEvents(t *testing.T) {
	t.Parallel()
	var events []string
	l := &Loop{onProgress: func(e string, _ map[string]any) { events = append(events, e) }}
	h := l.streamHandler(context.Background())
	h(llm.StreamDelta{ToolCall: &llm.ToolCallStreamDelta{Index: 0, Name: "search_code"}})
	h(llm.StreamDelta{ToolCall: &llm.ToolCallStreamDelta{Index: 0, Arguments: `{"q":"腾讯"}`}})
	want := []string{"tool_gen_start", "tool_gen_delta"}
	if len(events) != len(want) {
		t.Fatalf("events=%v want %v", events, want)
	}
	for i, e := range want {
		if events[i] != e {
			t.Fatalf("events=%v want %v", events, want)
		}
	}
}

func TestStreamHandlerThinkingStopsBeforeToolGen(t *testing.T) {
	t.Parallel()
	var events []string
	l := &Loop{onProgress: func(e string, _ map[string]any) { events = append(events, e) }}
	h := l.streamHandler(context.Background())
	h(llm.StreamDelta{ReasoningContent: "plan"})
	h(llm.StreamDelta{ToolCall: &llm.ToolCallStreamDelta{Index: 0, Name: "search_code"}})
	if events[0] != "thinking_start" || events[1] != "stream_delta" {
		t.Fatalf("prefix events=%v", events)
	}
	if events[2] != "thinking_stop" || events[3] != "tool_gen_start" {
		t.Fatalf("tool transition events=%v", events)
	}
}
