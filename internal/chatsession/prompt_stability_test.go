package chatsession_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestSystemPromptStableAcrossTurns(t *testing.T) {
	t.Parallel()
	stable := chatprompt.System()
	sess := &chatsession.ChatSession{
		ID:      "s1",
		Status:  "active",
		Messages: []llm.Message{{Role: llm.RoleSystem, Content: stable}},
	}
	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: "q1"})

	sess.Messages = append(sess.Messages, llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}}},
	})
	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleTool, ToolCallID: "c1", Content: `{"status":"ok","summary":"price=312"}`})

	sess.SyncChatSystemPrompt()
	if sess.Messages[0].Content != stable {
		t.Fatalf("system content changed after sync: %q", sess.Messages[0].Content)
	}

	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: "q2"})
	sess.SyncChatSystemPrompt()
	if sess.Messages[0].Content != stable {
		t.Fatalf("system content changed on turn 2: %q", sess.Messages[0].Content)
	}
	// Critical: system must not contain the injected dynamic context marker.
	if strings.Contains(sess.Messages[0].Content, "供参考，勿重复调用") {
		t.Fatal("system prompt leaked dynamic tool activity — prefix cache would break")
	}
}

func TestRuntimeMessagesInjectsContextBeforeLastUser(t *testing.T) {
	t.Parallel()
	sess := &chatsession.ChatSession{
		ID:     "s2",
		Status: "active",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "SYSTEM"},
			{Role: llm.RoleUser, Content: "q1"},
			{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
				{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
			}},
			{Role: llm.RoleTool, ToolCallID: "c1", Content: `{"summary":"price=312"}`},
			{Role: llm.RoleUser, Content: "q2"},
		},
	}
	out := sess.RuntimeMessages()
	if out[len(out)-1].Role != llm.RoleUser || out[len(out)-1].Content != "q2" {
		t.Fatalf("last message not q2: %+v", out[len(out)-1])
	}
	if out[len(out)-2].Role != llm.RoleUser || !strings.Contains(out[len(out)-2].Content, "Tool 活动") {
		t.Fatalf("context not injected before last user: %+v", out[len(out)-2])
	}
	if out[0].Content != "SYSTEM" {
		t.Fatalf("system changed: %q", out[0].Content)
	}
}

func TestRuntimeMessagesNoInjectionWhenNoActivity(t *testing.T) {
	t.Parallel()
	sess := &chatsession.ChatSession{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "SYSTEM"},
			{Role: llm.RoleUser, Content: "q1"},
		},
	}
	out := sess.RuntimeMessages()
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
}
