package chatsession_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestRefreshMetadataTitleUsesLatestUserMessage(t *testing.T) {
	t.Parallel()
	sess := &chatsession.ChatSession{ID: "chat-test"}
	sess.Messages = []llm.Message{
		{Role: llm.RoleUser, Content: "请只回复：测试成功"},
		{Role: llm.RoleAssistant, Content: "测试成功"},
		{Role: llm.RoleUser, Content: "再帮我分析一下小米"},
		{Role: llm.RoleAssistant, Content: "小米分析"},
	}
	sess.Title = "请只回复：测试成功"
	sess.RefreshMetadata()
	if sess.Title != "再帮我分析一下小米" {
		t.Fatalf("title=%q want latest user message", sess.Title)
	}
}
