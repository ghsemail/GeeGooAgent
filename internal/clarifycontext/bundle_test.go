package clarifycontext

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestBuildFromChatIncludesDialogueAndKind(t *testing.T) {
	chat := &chatsession.ChatSession{
		Summary: "用户关注腾讯走势",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "帮我分析一下腾讯的价格走势"},
			{Role: llm.RoleAssistant, Content: "## 腾讯走势\n- 偏弱"},
			{Role: llm.RoleUser, Content: "帮我看看哪些策略适合腾讯"},
		},
	}
	chat.SyncLastTurnPlan("dca_grid", "gather", "", false, nil)
	b := BuildFromChat(chat, "想了解哪种策略？", []string{"单指标信号", "组合信号（推荐）"}, "tool_clarify")
	if b.ClarifyKind != "strategy_family" {
		t.Fatalf("kind=%q", b.ClarifyKind)
	}
	if len(b.Dialogue) < 2 {
		t.Fatalf("dialogue=%v", b.Dialogue)
	}
	if b.LastTurn == nil || b.LastTurn.Domain != "dca_grid" {
		t.Fatalf("last_turn=%v", b.LastTurn)
	}
	raw, err := b.DecisionJSON()
	if err != nil || raw == "" {
		t.Fatalf("json err=%v len=%d", err, len(raw))
	}
}

func TestScrubMessageContentStripsThinking(t *testing.T) {
	in := "<think>hidden</think>可见正文"
	if got := scrubMessageContent(in); got != "可见正文" {
		t.Fatalf("got %q", got)
	}
}
