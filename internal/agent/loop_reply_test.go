package agent

import "testing"

func TestStripProviderNoiseThinkTags(t *testing.T) {
	in := "<think>hidden</think>visible"
	got := stripProviderNoise(in)
	if got != "visible" {
		t.Fatalf("got %q want visible", got)
	}
}

func TestStripProviderNoiseRedactedThinking(t *testing.T) {
	in := "<think>step</think>ok"
	got := stripProviderNoise(in)
	if got != "ok" {
		t.Fatalf("got %q want ok", got)
	}
}

func TestCleanAssistantVisibleTextExported(t *testing.T) {
	in := "<think>hidden</think>visible"
	got := CleanAssistantVisibleText(in)
	if got != "visible" {
		t.Fatalf("got %q want visible", got)
	}
}
