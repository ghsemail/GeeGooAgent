package llm

import "testing"

func TestVisibleAssistantContentStripsThinking(t *testing.T) {
	in := "hello <think>secret</think> world"
	got := VisibleAssistantContent(in, "")
	if got != "hello  world" {
		t.Fatalf("got=%q", got)
	}
}

func TestVisibleAssistantContentFallsBackToReasoning(t *testing.T) {
	got := VisibleAssistantContent("<think>x</think>", "fallback body")
	if got != "fallback body" {
		t.Fatalf("got=%q", got)
	}
}

func TestPublicAnswerContentNeverUsesReasoning(t *testing.T) {
	if got := PublicAnswerContent(""); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := PublicAnswerContent("最终中文"); got != "最终中文" {
		t.Fatalf("got=%q", got)
	}
}
