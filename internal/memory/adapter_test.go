package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
)

type stubSummarizer struct{ text string }

func (s stubSummarizer) Summarize(context.Context, []llm.Message, string, int) (string, error) {
	return s.text, nil
}

func TestAdapterCompressDelegatesToCompressor(t *testing.T) {
	t.Parallel()
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, ContextLength: 1000,
		ProtectFirstN: 1, ProtectLastN: 1,
	}
	compressor := prompt.NewCompressor(cfg, stubSummarizer{text: "summary"})
	ad := memory.NewAdapter(memory.AdapterConfig{Compressor: compressor})
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "sys"},
		{Role: llm.RoleUser, Content: "one"},
		{Role: llm.RoleAssistant, Content: "two"},
		{Role: llm.RoleUser, Content: "three"},
		{Role: llm.RoleAssistant, Content: "four"},
	}
	out, err := ad.Compress(context.Background(), memport.CompressInput{
		Messages: msgs, EstimatedTokens: 900,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.DidCompress {
		t.Fatalf("expected compress, got %+v", out)
	}
	if out.PreviousSummary != "summary" {
		t.Fatalf("summary=%q", out.PreviousSummary)
	}
}

func TestAdapterRecallSession(t *testing.T) {
	t.Parallel()
	store := infra.NewStateStore(t.TempDir())
	sessions := chatsession.NewChatSessionStore(store)
	s1, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	s1.Messages = append(s1.Messages,
		llm.Message{Role: llm.RoleUser, Content: "查腾讯股价"},
		llm.Message{Role: llm.RoleAssistant, Content: "ok", ToolCalls: []llm.ToolCall{
			{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
		}},
		llm.Message{Role: llm.RoleTool, ToolCallID: "c1", Content: `{"summary":"price=380","data":{"code":"00700.HK","price":380}}`},
	)
	s1.UpdatedAt = time.Now().UTC()
	if err := sessions.Save(s1); err != nil {
		t.Fatal(err)
	}

	ad := memory.NewAdapter(memory.AdapterConfig{Sessions: sessions})
	res, err := ad.Recall(context.Background(), memport.RecallQuery{
		Kind: memport.RecallSession, Query: "腾讯", Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) == 0 {
		t.Fatalf("expected hits, data=%+v", res.Data)
	}
}
