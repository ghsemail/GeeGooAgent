package runtimeapi

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestAssistantReplyForUserTurn_MultiStepReact(t *testing.T) {
	t.Parallel()
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "再帮我分析一下小米"},
		{Role: llm.RoleAssistant, Content: "", ReasoningContent: "planning"},
		{Role: llm.RoleTool, Content: `{"status":"ok"}`},
		{Role: llm.RoleAssistant, Content: "", ToolCalls: []llm.ToolCall{{Name: "search_code"}}},
		{Role: llm.RoleTool, Content: `{"status":"ok"}`},
		{Role: llm.RoleAssistant, Content: "以下是小米集团-W（**01810.HK**）的综合分析"},
	}
	got := assistantReplyForUserTurn(msgs, 0)
	want := "以下是小米集团-W（**01810.HK**）的综合分析"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildTurnsFromSession_LastAssistantReply(t *testing.T) {
	t.Parallel()
	sess := &chatsession.ChatSession{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "hello"},
			{Role: llm.RoleAssistant, Content: "你好"},
			{Role: llm.RoleUser, Content: "再帮我分析一下小米"},
			{Role: llm.RoleAssistant, Content: ""},
			{Role: llm.RoleTool, Content: `{}`},
			{Role: llm.RoleAssistant, Content: "小米分析结论"},
		},
		StepRecords: []chatsession.ChatStepRecord{
			{Kind: "plan"},
			{Kind: "reply"},
			{Kind: "plan"},
			{Kind: "tool", ToolName: "search_code", ToolStatus: "ok", Summary: "search_code"},
			{Kind: "reply"},
		},
	}
	turns := buildTurnsFromSession(sess)
	if len(turns) != 2 {
		t.Fatalf("turns=%d want 2", len(turns))
	}
	if got := turns[1]["reply"]; got != "小米分析结论" {
		t.Fatalf("reply=%v want 小米分析结论", got)
	}
}
