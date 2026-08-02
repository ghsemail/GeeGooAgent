package agent

import (
	"testing"
)

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

func TestFormatAssistantReplyForStorage_Passthrough(t *testing.T) {
	in := "以下是小米集团-W（**01810.HK**）的综合分析：---##📊小米集团-W价格趋势分析**当前价格**：**28.78港元**"
	got := formatAssistantReplyForStorage(in)
	if got != in {
		t.Fatalf("storage should keep model markdown as-is, got %q", got)
	}
}
