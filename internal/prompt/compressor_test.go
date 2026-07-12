package prompt

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type scriptedSummarizer struct {
	text    string
	err     error
	sawPrev string
}

func (s *scriptedSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	s.sawPrev = previousSummary
	if s.err != nil {
		return "", s.err
	}
	return s.text, nil
}

func TestEstimateTokensRough(t *testing.T) {
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "abcd"}} // 4 chars → 1 token
	if got := EstimateTokens(msgs); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestShouldCompress(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.5, HygieneThreshold: 0.85, ContextLength: 1000,
		ProtectFirstN: 3, ProtectLastN: 2,
	}
	c := NewCompressor(cfg, nil)
	msgs := make([]llm.Message, 10)
	for i := range msgs {
		msgs[i] = llm.Message{Role: llm.RoleUser, Content: "x"}
	}
	if !c.ShouldCompress(600, len(msgs)) {
		t.Fatal("want compress")
	}
	if c.ShouldCompress(400, len(msgs)) {
		t.Fatal("below threshold")
	}
	if c.ShouldHygiene(600, len(msgs)) {
		t.Fatal("600 < 85% of 1000")
	}
	if !c.ShouldHygiene(900, len(msgs)) {
		t.Fatal("want hygiene at 90%")
	}
	cfg.Enabled = false
	c = NewCompressor(cfg, nil)
	if c.ShouldCompress(9999, len(msgs)) {
		t.Fatal("disabled")
	}
}

func TestClearOldToolResults(t *testing.T) {
	cfg := config.ResolvedCompression{ClearToolMinChars: 200, ProtectLastN: 2, ProtectFirstN: 1}
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "sys"},
		{Role: llm.RoleTool, Content: strings.Repeat("a", 250)},
		{Role: llm.RoleTool, Content: "short"},
		{Role: llm.RoleUser, Content: "tail1"},
		{Role: llm.RoleUser, Content: "tail2"},
	}
	out := clearOldToolResults(msgs, 3, cfg.ClearToolMinChars) // protect from index 3
	if out[1].Content == strings.Repeat("a", 250) {
		t.Fatal("large tool outside tail should clear")
	}
	if out[2].Content != "short" {
		t.Fatal("short tool kept")
	}
	if out[3].Content != "tail1" {
		t.Fatal("tail untouched")
	}
}

func TestAlignBoundaryBackward(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "s"},
		{Role: llm.RoleUser, Content: "u1"},
		{Role: llm.RoleAssistant, Content: "", ToolCalls: []llm.ToolCall{{ID: "1", Name: "t"}}},
		{Role: llm.RoleTool, ToolCallID: "1", Content: "r"},
		{Role: llm.RoleUser, Content: "u2"},
	}
	got := alignBoundaryBackward(msgs, 3)
	if got != 2 {
		t.Fatalf("got %d want 2", got)
	}
}

func TestSanitizeToolPairs(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "a", Name: "x"}}},
		{Role: llm.RoleUser, Content: "hi"},
		{Role: llm.RoleTool, ToolCallID: "orphan", Content: "x"},
	}
	out := sanitizeToolPairs(msgs)
	hasStub := false
	for _, m := range out {
		if m.Role == llm.RoleTool && m.ToolCallID == "a" {
			hasStub = true
		}
		if m.ToolCallID == "orphan" {
			t.Fatal("orphan tool should be removed")
		}
	}
	if !hasStub {
		t.Fatal("missing stub for tool_call a")
	}
}

func TestDetermineCut(t *testing.T) {
	cfg := config.ResolvedCompression{
		ProtectFirstN: 2, ProtectLastN: 2,
		ContextLength: 1000, Threshold: 0.5, TargetRatio: 0.2,
	}
	var msgs []llm.Message
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: "head0"})
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: "head1"})
	for i := 0; i < 10; i++ {
		msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: strings.Repeat("m", 200)})
	}
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: "tail0"})
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: "tail1"})

	headEnd, cut := determineCut(msgs, cfg)
	if headEnd != 2 {
		t.Fatalf("headEnd=%d want 2", headEnd)
	}
	if cut <= headEnd {
		t.Fatalf("cut=%d should leave middle after headEnd=%d", cut, headEnd)
	}
	if len(msgs)-cut < cfg.ProtectLastN {
		t.Fatalf("tail too short: cut=%d len=%d", cut, len(msgs))
	}
}

func TestCompressSuccess(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, TargetRatio: 0.2,
		ProtectFirstN: 2, ProtectLastN: 2, ContextLength: 1000, ClearToolMinChars: 50,
	}
	sum := &scriptedSummarizer{text: "## Goal\nTest"}
	c := NewCompressor(cfg, sum)
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "SYSTEM_STABLE"},
		{Role: llm.RoleUser, Content: "first"},
		{Role: llm.RoleAssistant, Content: "mid1"},
		{Role: llm.RoleUser, Content: "mid2"},
		{Role: llm.RoleAssistant, Content: "mid3"},
		{Role: llm.RoleUser, Content: "tail-a"},
		{Role: llm.RoleUser, Content: "tail-b"},
	}
	out, did, newSummary, err := c.Compress(context.Background(), msgs, "", 9999)
	if err != nil || !did {
		t.Fatalf("did=%v err=%v", did, err)
	}
	if newSummary != "## Goal\nTest" {
		t.Fatalf("newSummary=%q", newSummary)
	}
	if out[0].Content != "SYSTEM_STABLE" {
		t.Fatal("system must stay identical")
	}
	joined := ""
	for _, m := range out {
		joined += m.Content
	}
	if !strings.Contains(joined, "## Goal") {
		t.Fatal("summary missing")
	}
	if len(out) >= len(msgs) {
		t.Fatalf("expected fewer msgs, before=%d after=%d", len(msgs), len(out))
	}
}

func TestCompressSummaryFailureSkips(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 100,
	}
	c := NewCompressor(cfg, &scriptedSummarizer{err: errors.New("boom")})
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "s"},
		{Role: llm.RoleUser, Content: "a"},
		{Role: llm.RoleUser, Content: "b"},
		{Role: llm.RoleUser, Content: "c"},
	}
	out, did, newSummary, err := c.Compress(context.Background(), msgs, "keep-me", 9999)
	if err != nil {
		t.Fatal(err)
	}
	if did {
		t.Fatal("should skip")
	}
	if newSummary != "keep-me" {
		t.Fatalf("newSummary=%q", newSummary)
	}
	if len(out) != len(msgs) {
		t.Fatal("messages must be unchanged")
	}
}

func TestCompressPassesPreviousSummary(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 100,
	}
	sum := &scriptedSummarizer{text: "updated"}
	c := NewCompressor(cfg, sum)
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "s"},
		{Role: llm.RoleUser, Content: "a"},
		{Role: llm.RoleUser, Content: "b"},
		{Role: llm.RoleUser, Content: "c"},
	}
	_, did, newSummary, err := c.Compress(context.Background(), msgs, "old-summary", 9999)
	if err != nil || !did {
		t.Fatalf("did=%v err=%v", did, err)
	}
	if sum.sawPrev != "old-summary" {
		t.Fatalf("prev=%q", sum.sawPrev)
	}
	if newSummary != "updated" {
		t.Fatalf("newSummary=%q", newSummary)
	}
}

func TestCompressUsesProvidedTokenEstimate(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.5, TargetRatio: 0.01,
		ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 1000,
	}
	sum := &scriptedSummarizer{text: "summary"}
	c := NewCompressor(cfg, sum)
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "s"},
	}
	for i := 0; i < 10; i++ {
		msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: strings.Repeat("x", 20)})
	}
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: "tail"})
	if EstimateTokens(msgs) >= 500 {
		t.Fatal("test requires local estimate below threshold")
	}
	out, did, newSummary, err := c.Compress(context.Background(), msgs, "", 600)
	if err != nil || !did {
		t.Fatalf("did=%v err=%v", did, err)
	}
	if newSummary != "summary" {
		t.Fatalf("newSummary=%q", newSummary)
	}
	if len(out) >= len(msgs) {
		t.Fatalf("expected compression, before=%d after=%d", len(msgs), len(out))
	}
}
