package memory_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
)

type stubSummarizer struct{ text string }

func (s stubSummarizer) Summarize(context.Context, []llm.Message, string, int) (string, error) {
	return s.text, nil
}

type fakeFactsStore struct {
	rows []facts.Row
}

func (f *fakeFactsStore) SearchRows(_ context.Context, _, query string, topK int) ([]facts.Row, error) {
	if topK <= 0 {
		topK = len(f.rows)
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]facts.Row, 0, len(f.rows))
	for _, r := range f.rows {
		if q == "" || strings.Contains(strings.ToLower(r.Subject), q) || strings.Contains(strings.ToLower(r.Content), q) {
			out = append(out, r)
		}
	}
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
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

func TestAdapterRecallFacts(t *testing.T) {
	t.Parallel()
	ad := memory.NewAdapter(memory.AdapterConfig{
		Facts: &fakeFactsStore{rows: []facts.Row{
			{ID: 1, Subject: "tencent", Content: "user tracks 00700.HK price"},
		}},
	})
	res, err := ad.Recall(context.Background(), memport.RecallQuery{
		Kind: memport.RecallSession, Query: "tencent", Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) == 0 {
		t.Fatalf("expected hits, data=%+v", res.Data)
	}
	if res.Data["recall_source"] != "facts" {
		t.Fatalf("source=%v", res.Data["recall_source"])
	}
}

func TestAdapterRecallFactsRanker(t *testing.T) {
	t.Parallel()
	store := &fakeFactsStore{rows: []facts.Row{
		{ID: 1, Subject: "maotai", Content: "tracks 600519"},
		{ID: 2, Subject: "tencent", Content: "tracks 00700"},
	}}
	reverse := func(_ context.Context, hits []memport.RecallHit) ([]memport.RecallHit, error) {
		out := make([]memport.RecallHit, len(hits))
		for i, h := range hits {
			out[len(hits)-1-i] = h
		}
		return out, nil
	}
	ad := memory.NewAdapter(memory.AdapterConfig{Facts: store, SessionRanker: reverse})
	res, err := ad.Recall(context.Background(), memport.RecallQuery{
		Kind: memport.RecallSession, Query: "", Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) < 2 {
		t.Fatalf("expected 2+ hits, got %d", len(res.Hits))
	}
	adPlain := memory.NewAdapter(memory.AdapterConfig{Facts: store})
	plain, err := adPlain.Recall(context.Background(), memport.RecallQuery{
		Kind: memport.RecallSession, Query: "", Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Hits) < 2 {
		t.Fatal("plain recall")
	}
	if res.Hits[0].ID == plain.Hits[0].ID {
		t.Fatalf("ranker should change order: ranked=%s plain=%s", res.Hits[0].ID, plain.Hits[0].ID)
	}
}
