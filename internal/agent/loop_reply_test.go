package agent

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
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

func TestFormatAssistantReplyForStorage_NormalizesGlue(t *testing.T) {
	in := "以下是小米集团-W（**01810.HK**）的综合分析：---##📊小米集团-W价格趋势分析**当前价格**：**28.78港元**"
	got := formatAssistantReplyForStorage(in)
	if got == in {
		t.Fatalf("storage should normalize glued markdown, got unchanged %q", got)
	}
	if !strings.Contains(got, "##") {
		t.Fatalf("expected heading split, got %q", got)
	}
}

func TestWithReplyFormatReminder(t *testing.T) {
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "分析美团"}}
	got := withReplyFormatReminder(msgs, 0)
	if len(got) != 1 {
		t.Fatalf("round 0 should not append reminder, got %d messages", len(got))
	}
	got = withReplyFormatReminder(msgs, 1)
	if len(got) != 2 {
		t.Fatalf("round 1 should append reminder, got %d messages", len(got))
	}
	if !strings.Contains(got[1].Content, "[FORMAT]") {
		t.Fatalf("missing format reminder: %q", got[1].Content)
	}
}
