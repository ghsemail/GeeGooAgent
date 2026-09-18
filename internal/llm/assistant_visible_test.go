package llm

import "testing"

func TestVisibleAssistantContentStripsThinking(t *testing.T) {
	in := "<think>secret plan</think>\n\n## 一句话定位\n\nhello"
	got := VisibleAssistantContent(in, "")
	if got != "## 一句话定位\n\nhello" {
		t.Fatalf("got=%q", got)
	}
}

func TestVisibleAssistantContentFallsBackToReasoning(t *testing.T) {
	got := VisibleAssistantContent("<think>x</think>", "fallback body")
	if got != "fallback body" {
		t.Fatalf("got=%q", got)
	}
}
