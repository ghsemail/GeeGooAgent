package llm

import (
	"strings"
	"testing"
)

// Production evidence (PR#70 deploy, MiniMax-M3 + TaskSynthesis, 2026-10-22):
// - compose_model=MiniMax-M3, max_tokens=2048, all 6 runs deadline_hit ~121s
// - eval_judgment began with bare English in delta.content (no separate reasoning_content):
//   "need answer onlyfinalChinese body, formal ..."
// - compose_reasoning_debug empty (ReasoningContent channel unused)
// - Chinese appeared only after ~780+ chars of English planning in content
//
// Tagged variant (same provider family, chat/ReAct path): reasoning inside
// <think>...</think> in content — see TestAnswerStreamAccumulatorTaggedThinking.

func TestMiniMaxBareEnglishInContentEvidence(t *testing.T) {
	// Simulated stream chunks matching production shape (content channel only).
	deltas := []string{
		"need answer only",
		"finalChinese body, formal ",
		"... Need parse details ",
		"probe metrics ",
		"信号诊断：",
		"该策略在 probe 阶段表现",
	}
	var acc AnswerStreamAccumulator
	for _, d := range deltas {
		acc.PushContent(d)
	}
	got := acc.Visible()
	if !strings.Contains(got, "need answer") || !strings.Contains(got, "信号诊断") {
		t.Fatalf("evidence stream shape: %q", got)
	}
	// Untagged English remains in visible — fix is TaskClassify/chat compose model + ThinkingOff, not tag stripping alone.
	if acc.ThinkingExtracted() != "" {
		t.Fatalf("unexpected thinking extract: %q", acc.ThinkingExtracted())
	}
}

func TestAnswerStreamAccumulatorTaggedThinking(t *testing.T) {
	deltas := []string{
		"prefix ",
		"<think>english plan</think>",
		"中文正文",
	}
	var acc AnswerStreamAccumulator
	var visible, thinking strings.Builder
	acc.OnVisible = func(chunk string) { visible.WriteString(chunk) }
	acc.OnThinking = func(chunk string) { thinking.WriteString(chunk) }
	for _, d := range deltas {
		acc.PushContent(d)
	}
	if strings.Contains(visible.String(), "english plan") {
		t.Fatalf("thinking leaked: %q", visible.String())
	}
	if !strings.Contains(thinking.String(), "english plan") {
		t.Fatalf("thinking debug empty: %q", thinking.String())
	}
	if visible.String() != "prefix 中文正文" {
		t.Fatalf("visible=%q", visible.String())
	}
}

func TestAnswerStreamAccumulatorUnclosedThinkingTag(t *testing.T) {
	deltas := []string{"<think>wip plan", " still planning", "</think>答案"}
	var acc AnswerStreamAccumulator
	var visible, thinking strings.Builder
	acc.OnVisible = func(chunk string) { visible.WriteString(chunk) }
	acc.OnThinking = func(chunk string) { thinking.WriteString(chunk) }
	for _, d := range deltas {
		acc.PushContent(d)
	}
	if visible.String() != "答案" {
		t.Fatalf("visible=%q", visible.String())
	}
	if !strings.Contains(thinking.String(), "wip plan") {
		t.Fatalf("thinking=%q", thinking.String())
	}
}
