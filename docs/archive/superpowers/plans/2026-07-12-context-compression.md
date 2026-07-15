# Context Compression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Hermes-style token-threshold context compression to GeeGooAgent’s ReAct loop, with configurable auxiliary summarizer and write-back to the session.

**Architecture:** New `internal/prompt` package owns estimate + four-phase `Compressor`. `ReActLoop` calls it before each `gateway.Chat`, updates `session.Messages`, and records `lastPromptTokens` from API usage. `App` builds the compressor from `compression` / `auxiliary.compression` config and wires it onto `Agent.Loop` (the path chat/runtime actually use).

**Tech Stack:** Go 1.x, existing `internal/llm` (Provider/Gateway/MockProvider), `go test`.

**Spec:** `docs/superpowers/specs/2026-07-12-context-compression-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `internal/config/config.go` | `CompressionConfig`, `AuxiliaryLLMConfig`, defaults via `EffectiveCompression` / `EffectiveAuxiliaryCompression` |
| `internal/config/config_test.go` | Defaults + JSON load |
| `internal/prompt/estimate.go` | `EstimateTokens` |
| `internal/prompt/compressor.go` | `Compressor`, phases 1/2/4, `ShouldCompress`, `Compress` |
| `internal/prompt/summary.go` | `Summarizer` interface + `ProviderSummarizer` |
| `internal/prompt/compressor_test.go` | Unit tests (mock summarizer) |
| `internal/llm/mock.go` | Optional `Err` field for failure scripts |
| `internal/runtime/react.go` | Hook + `lastPromptTokens` + `SetCompressor` |
| `internal/runtime/react_compress_test.go` | Loop-level compress smoke test |
| `internal/app/app.go` | Build compressor; wire Agent.Loop; fix RebuildGateway to update Agent |
| `docs/architecture/overview.md` | Mention compressor |
| `docs/architecture/layers/L3-memory/compaction.md` | Point to new spec |
| `deploy/hermes-parity-comparison.md` | Mark compression ✅ |

---

### Task 1: Compression + auxiliary config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/config/config_test.go`:

```go
func TestEffectiveCompressionDefaults(t *testing.T) {
	cfg := &AppConfig{}
	c := cfg.EffectiveCompression()
	if !c.Enabled {
		t.Fatal("enabled default true")
	}
	if c.Threshold != 0.5 || c.TargetRatio != 0.2 || c.ProtectLastN != 20 {
		t.Fatalf("defaults: %+v", c)
	}
	if c.ProtectFirstN != 3 || c.ContextLength != 128000 || c.ClearToolMinChars != 200 {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestLoadCompressionJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk",
		"compression": {"enabled": false, "threshold": 0.6, "context_length": 64000},
		"auxiliary": {"compression": {"provider": "deepseek", "model": "deepseek-chat"}}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Compression.Enabled == nil || *cfg.Compression.Enabled {
		t.Fatal("want enabled=false")
	}
	c := cfg.EffectiveCompression()
	if c.Enabled || c.Threshold != 0.6 || c.ContextLength != 64000 {
		t.Fatalf("got %+v", c)
	}
	aux := cfg.EffectiveAuxiliaryCompression()
	if aux.Provider != "deepseek" || aux.Model != "deepseek-chat" {
		t.Fatalf("aux %+v", aux)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd D:\Geegoo\GeeGooAgent
go test ./internal/config/ -run 'TestEffectiveCompressionDefaults|TestLoadCompressionJSON' -count=1
```

Expected: compile error / undefined `EffectiveCompression`.

- [ ] **Step 3: Implement config types**

In `internal/config/config.go`, add types and wire into `AppConfig`:

```go
// CompressionConfig is the JSON shape (pointers allow distinguishing unset).
type CompressionConfig struct {
	Enabled           *bool    `json:"enabled,omitempty"`
	Threshold         float64  `json:"threshold,omitempty"`
	TargetRatio       float64  `json:"target_ratio,omitempty"`
	ProtectLastN      int      `json:"protect_last_n,omitempty"`
	ProtectFirstN     int      `json:"protect_first_n,omitempty"`
	ContextLength     int      `json:"context_length,omitempty"`
	ClearToolMinChars int      `json:"clear_tool_min_chars,omitempty"`
}

// AuxiliaryLLMConfig is optional summarizer credentials.
type AuxiliaryLLMConfig struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	TokenKey string `json:"token_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type AuxiliaryConfig struct {
	Compression AuxiliaryLLMConfig `json:"compression"`
}

// On AppConfig add:
//   Compression CompressionConfig `json:"compression"`
//   Auxiliary   AuxiliaryConfig   `json:"auxiliary"`
```

Add resolved value type + helpers:

```go
// ResolvedCompression is EffectiveCompression output (no pointers).
type ResolvedCompression struct {
	Enabled           bool
	Threshold         float64
	TargetRatio       float64
	ProtectLastN      int
	ProtectFirstN     int
	ContextLength     int
	ClearToolMinChars int
}

func (c *AppConfig) EffectiveCompression() ResolvedCompression {
	out := ResolvedCompression{
		Enabled: true, Threshold: 0.5, TargetRatio: 0.2,
		ProtectLastN: 20, ProtectFirstN: 3, ContextLength: 128000, ClearToolMinChars: 200,
	}
	if c == nil {
		return out
	}
	src := c.Compression
	if src.Enabled != nil {
		out.Enabled = *src.Enabled
	}
	if src.Threshold > 0 {
		out.Threshold = src.Threshold
	}
	if src.TargetRatio > 0 {
		out.TargetRatio = src.TargetRatio
	}
	if src.ProtectLastN > 0 {
		out.ProtectLastN = src.ProtectLastN
	}
	if src.ProtectFirstN > 0 {
		out.ProtectFirstN = src.ProtectFirstN
	}
	if src.ContextLength > 0 {
		out.ContextLength = src.ContextLength
	}
	if src.ClearToolMinChars > 0 {
		out.ClearToolMinChars = src.ClearToolMinChars
	}
	if out.Threshold > 1 {
		out.Threshold = 1
	}
	if out.TargetRatio < 0.1 {
		out.TargetRatio = 0.1
	}
	if out.TargetRatio > 0.8 {
		out.TargetRatio = 0.8
	}
	return out
}

// EffectiveAuxiliaryCompression returns aux fields with empty → main LLM fallback.
func (c *AppConfig) EffectiveAuxiliaryCompression() AuxiliaryLLMConfig {
	var aux AuxiliaryLLMConfig
	if c != nil {
		aux = c.Auxiliary.Compression
	}
	if c == nil {
		return aux
	}
	if strings.TrimSpace(aux.Provider) == "" {
		aux.Provider = c.LLM.Provider
	}
	if strings.TrimSpace(aux.Model) == "" {
		aux.Model = c.LLM.Model
	}
	if strings.TrimSpace(aux.TokenKey) == "" {
		aux.TokenKey = c.LLM.TokenKey
	}
	if strings.TrimSpace(aux.BaseURL) == "" {
		aux.BaseURL = c.LLM.BaseURL
	}
	return aux
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/config/ -run 'TestEffectiveCompressionDefaults|TestLoadCompressionJSON' -count=1
```

- [ ] **Step 5: Commit** (only if user asked to commit; otherwise skip and continue)

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "$(cat <<'EOF'
feat(config): add compression and auxiliary summarizer settings

EOF
)"
```

---

### Task 2: EstimateTokens + ShouldCompress

**Files:**
- Create: `internal/prompt/estimate.go`
- Create: `internal/prompt/compressor.go` (skeleton)
- Create: `internal/prompt/compressor_test.go`

- [ ] **Step 1: Write failing tests**

```go
package prompt

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestEstimateTokensRough(t *testing.T) {
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "abcd"}} // 4 chars → 1 token
	if got := EstimateTokens(msgs); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestShouldCompress(t *testing.T) {
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.5, ContextLength: 1000,
		ProtectFirstN: 3, ProtectLastN: 2,
	}
	c := NewCompressor(cfg, nil)
	// 10 msgs → enough middle; tokens 600 >= 500
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
	cfg.Enabled = false
	c = NewCompressor(cfg, nil)
	if c.ShouldCompress(9999, len(msgs)) {
		t.Fatal("disabled")
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/prompt/ -count=1
```

- [ ] **Step 3: Implement**

`internal/prompt/estimate.go`:

```go
package prompt

import "github.com/ghsemail/GeeGooAgent/internal/llm"

// EstimateTokens rough-estimates prompt size as sum(len(content))/4 (+ tool call names).
func EstimateTokens(messages []llm.Message) int {
	n := 0
	for _, m := range messages {
		n += len(m.Content)
		for _, tc := range m.ToolCalls {
			n += len(tc.Name) + 16
		}
	}
	if n <= 0 {
		return 0
	}
	return (n + 3) / 4
}
```

In `compressor.go` (partial):

```go
package prompt

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type Compressor struct {
	cfg             config.ResolvedCompression
	summarizer      Summarizer // defined in Task 4; for now use interface in summary.go stub
	previousSummary string
}

func NewCompressor(cfg config.ResolvedCompression, sum Summarizer) *Compressor {
	return &Compressor{cfg: cfg, summarizer: sum}
}

func (c *Compressor) ShouldCompress(tokenEstimate, messageCount int) bool {
	if c == nil || !c.cfg.Enabled {
		return false
	}
	minMsgs := c.cfg.ProtectFirstN + c.cfg.ProtectLastN + 1
	if messageCount < minMsgs {
		return false
	}
	thresholdTokens := int(float64(c.cfg.ContextLength) * c.cfg.Threshold)
	return tokenEstimate >= thresholdTokens
}
```

Create `summary.go` stub so it compiles:

```go
package prompt

import (
	"context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// Summarizer produces a structured compaction summary.
type Summarizer interface {
	Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error)
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/prompt/ -count=1
```

- [ ] **Step 5: Commit** (if requested)

---

### Task 3: Phase 1 / 2 / 4 (no LLM)

**Files:**
- Modify: `internal/prompt/compressor.go`
- Modify: `internal/prompt/compressor_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
	// cut proposed at tool index 3 → align to assistant index 2
	got := alignBoundaryBackward(msgs, 3)
	if got != 2 {
		t.Fatalf("got %d want 2", got)
	}
}

func TestSanitizeToolPairs(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "a", Name: "x"}}},
		// missing tool result
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
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement helpers in `compressor.go`**

```go
const clearedToolPlaceholder = "[Old tool output cleared to save context space]"

func clearOldToolResults(msgs []llm.Message, protectFrom, minChars int) []llm.Message {
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	for i := 0; i < protectFrom && i < len(out); i++ {
		if out[i].Role == llm.RoleTool && len(out[i].Content) > minChars {
			out[i].Content = clearedToolPlaceholder
		}
	}
	return out
}

func alignBoundaryBackward(msgs []llm.Message, cut int) int {
	if cut <= 0 || cut >= len(msgs) {
		return cut
	}
	i := cut
	for i > 0 && msgs[i].Role == llm.RoleTool {
		i--
	}
	if i >= 0 && i < len(msgs) && msgs[i].Role == llm.RoleAssistant && len(msgs[i].ToolCalls) > 0 {
		return i
	}
	return cut
}

func sanitizeToolPairs(msgs []llm.Message) []llm.Message {
	need := map[string]bool{}
	have := map[string]bool{}
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				need[tc.ID] = true
			}
		}
		if m.Role == llm.RoleTool && m.ToolCallID != "" {
			have[m.ToolCallID] = true
		}
	}
	var out []llm.Message
	for _, m := range msgs {
		if m.Role == llm.RoleTool {
			if m.ToolCallID == "" || !need[m.ToolCallID] {
				continue // orphan
			}
			out = append(out, m)
			continue
		}
		out = append(out, m)
		if m.Role == llm.RoleAssistant {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" && !have[tc.ID] {
					out = append(out, llm.Message{
						Role: llm.RoleTool, ToolCallID: tc.ID,
						Content: `{"status":"skipped","summary":"[tool result omitted during compaction]"}`,
					})
					have[tc.ID] = true
				}
			}
		}
	}
	return out
}

// determineCut returns index where tail starts (middle is [headEnd, cut)).
func determineCut(msgs []llm.Message, cfg config.ResolvedCompression) (headEnd, cut int) {
	headEnd = cfg.ProtectFirstN
	if headEnd > len(msgs) {
		headEnd = len(msgs)
	}
	thresholdTokens := int(float64(cfg.ContextLength) * cfg.Threshold)
	budget := int(float64(thresholdTokens) * cfg.TargetRatio)
	acc := 0
	cut = len(msgs)
	for i := len(msgs) - 1; i >= headEnd; i-- {
		acc += EstimateTokens([]llm.Message{msgs[i]})
		cut = i
		if acc >= budget {
			break
		}
	}
	if len(msgs)-cut < cfg.ProtectLastN {
		cut = len(msgs) - cfg.ProtectLastN
		if cut < headEnd {
			cut = headEnd
		}
	}
	cut = alignBoundaryBackward(msgs, cut)
	if cut < headEnd {
		cut = headEnd
	}
	return headEnd, cut
}
```

Also export a testable `CompressDry` path later via full `Compress` in Task 4.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/prompt/ -count=1
```

- [ ] **Step 5: Commit** (if requested)

---

### Task 4: Phase 3 summary + full Compress

**Files:**
- Modify: `internal/prompt/summary.go`
- Modify: `internal/prompt/compressor.go`
- Modify: `internal/prompt/compressor_test.go`
- Modify: `internal/llm/mock.go` (add `Err error` optional)

- [ ] **Step 1: Extend MockProvider**

```go
// In MockProvider:
Err error

func (m *MockProvider) Chat(...) (*Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	// existing logic
}
```

- [ ] **Step 2: Write Compress tests**

```go
type scriptedSummarizer struct {
	text string
	err  error
	sawPrev string
}

func (s *scriptedSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	s.sawPrev = previousSummary
	if s.err != nil {
		return "", s.err
	}
	return s.text, nil
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
	out, did, err := c.Compress(context.Background(), msgs)
	if err != nil || !did {
		t.Fatalf("did=%v err=%v", did, err)
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
	out, did, err := c.Compress(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if did {
		t.Fatal("should skip")
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
	c.previousSummary = "old-summary"
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "s"},
		{Role: llm.RoleUser, Content: "a"},
		{Role: llm.RoleUser, Content: "b"},
		{Role: llm.RoleUser, Content: "c"},
	}
	_, _, _ = c.Compress(context.Background(), msgs)
	if sum.sawPrev != "old-summary" {
		t.Fatalf("prev=%q", sum.sawPrev)
	}
}
```

- [ ] **Step 3: Implement `ProviderSummarizer` + `Compress`**

`summary.go`:

```go
const summarySystem = `You compress prior conversation turns into a structured brief for a stock-analysis agent.
Fill these sections (use Chinese or English to match the source):
## Goal
## Constraints & Preferences
## Progress
### Done
### In Progress
### Blocked
## Key Decisions
## Relevant Symbols / Reports
## Next Steps
## Critical Context
If an earlier summary is provided, UPDATE it instead of rewriting from scratch.`

type ProviderSummarizer struct {
	Provider llm.Provider
}

func (p *ProviderSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	if p == nil || p.Provider == nil {
		return "", fmt.Errorf("summarizer provider nil")
	}
	var b strings.Builder
	if previousSummary != "" {
		b.WriteString("Previous summary:\n")
		b.WriteString(previousSummary)
		b.WriteString("\n\n")
	}
	b.WriteString("Turns to compress:\n")
	for _, m := range middle {
		b.WriteString(string(m.Role))
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: summarySystem},
		{Role: llm.RoleUser, Content: b.String()},
	}
	resp, err := p.Provider.Chat(ctx, msgs, nil, 0.2, maxTokens)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(resp.Content)
	if text == "" {
		return "", fmt.Errorf("empty summary")
	}
	return text, nil
}
```

`Compress` method:

```go
// Compress returns (messages, didCompress, err).
// On summarizer failure: returns original messages, did=false, err=nil.
func (c *Compressor) Compress(ctx context.Context, messages []llm.Message) ([]llm.Message, bool, error) {
	if c == nil || !c.ShouldCompress(EstimateTokens(messages), len(messages)) {
		return messages, false, nil
	}
	headEnd, cut := determineCut(messages, c.cfg)
	if cut <= headEnd {
		return messages, false, nil
	}
	working := clearOldToolResults(messages, cut, c.cfg.ClearToolMinChars)
	head := working[:headEnd]
	middle := working[headEnd:cut]
	tail := working[cut:]

	contentTokens := EstimateTokens(middle)
	maxTok := contentTokens * 20 / 100
	if maxTok < 2000 {
		maxTok = 2000
	}
	capTok := c.cfg.ContextLength * 5 / 100
	if capTok > 12000 {
		capTok = 12000
	}
	if maxTok > capTok {
		maxTok = capTok
	}

	if c.summarizer == nil {
		return messages, false, nil
	}
	summary, err := c.summarizer.Summarize(ctx, middle, c.previousSummary, maxTok)
	if err != nil {
		return messages, false, nil // skip — conservative
	}
	c.previousSummary = summary

	summaryMsg := llm.Message{
		Role: llm.RoleUser,
		Content: "[CONTEXT COMPACTION] Earlier turns were compacted into the following summary:\n" + summary,
	}
	// Avoid consecutive same role if head ends with user.
	if len(head) > 0 && head[len(head)-1].Role == llm.RoleUser {
		summaryMsg.Role = llm.RoleAssistant
	}

	out := make([]llm.Message, 0, len(head)+1+len(tail))
	out = append(out, head...)
	out = append(out, summaryMsg)
	out = append(out, tail...)
	out = sanitizeToolPairs(out)
	return out, true, nil
}

func (c *Compressor) PreviousSummary() string {
	if c == nil {
		return ""
	}
	return c.previousSummary
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/prompt/ -count=1
```

- [ ] **Step 5: Commit** (if requested)

---

### Task 5: Wire into ReActLoop

**Files:**
- Modify: `internal/runtime/react.go`
- Create: `internal/runtime/react_compress_test.go`

- [ ] **Step 1: Write loop test**

```go
func TestRunTurnCompressesBeforeChat(t *testing.T) {
	// Build a compressor with tiny threshold + scripted summarizer (duplicate small stub in test file
	// or export test helper). Use prompt.NewCompressor with mock summarizer.
	// Mock LLM returns one final text reply (no tools).
	// Session starts with many large messages so ShouldCompress is true.
	// After RunTurn, session.Messages length should drop and contain compaction marker.
}
```

Concrete test body:

```go
package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type stubSum struct{ text string }

func (s stubSum) Summarize(ctx context.Context, middle []llm.Message, prev string, maxTokens int) (string, error) {
	return s.text, nil
}

func TestRunTurnCompressesBeforeChat(t *testing.T) {
	mock := &llm.MockProvider{Responses: []*llm.Response{
		{Content: "final answer", Usage: llm.TokenUsage{PromptTokens: 50}},
	}}
	gw := llm.NewGateway(mock, llm.GatewayConfig{MaxRetries: 1, RetryWait: 0, MaxTokens: 128})
	gw.SetSleep(func(d time.Duration) {})
	reg := tools.NewRegistry()
	exec := NewExecutor(reg)
	loop := NewReActLoop(gw, exec)
	cfg := config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, ProtectFirstN: 2, ProtectLastN: 2,
		ContextLength: 100, ClearToolMinChars: 10,
	}
	loop.SetCompressor(prompt.NewCompressor(cfg, stubSum{text: "## Goal\nok"}))

	sess := NewSession("t1")
	// system already present; add many mid messages
	for i := 0; i < 8; i++ {
		sess.AppendMessage(llm.Message{Role: llm.RoleUser, Content: strings.Repeat("m", 80)})
		sess.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: strings.Repeat("a", 80)})
	}
	before := len(sess.Messages)
	res := loop.RunTurn(context.Background(), sess, "latest question", tools.Context{}, nil)
	if res.Failed {
		t.Fatalf("%s", res.Error)
	}
	if len(sess.Messages) >= before+1 {
		// +1 for new user; compression should still shrink overall vs uncompressed growth
		t.Logf("before=%d after=%d", before, len(sess.Messages))
	}
	found := false
	for _, m := range sess.Messages {
		if strings.Contains(m.Content, "CONTEXT COMPACTION") || strings.Contains(m.Content, "## Goal") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected compaction summary in session")
	}
	if sess.Messages[0].Role != llm.RoleSystem {
		t.Fatal("system must remain first")
	}
}
```

Adjust assertion: after compress + new user + assistant reply, must contain compaction text.

- [ ] **Step 2: Run — expect FAIL** (SetCompressor undefined)

- [ ] **Step 3: Implement hook in `react.go`**

Add fields + setters:

```go
type ReActLoop struct {
	gateway          *llm.Gateway
	executor         *Executor
	maxToolRounds    int
	onProgress       ProgressFunc
	compressor       *prompt.Compressor
	lastPromptTokens int
}

func (l *ReActLoop) SetCompressor(c *prompt.Compressor) {
	l.compressor = c
}
```

Before `l.gateway.Chat(...)`:

```go
messages = l.applyCompression(ctx, session, messages)
```

After successful Chat:

```go
if resp.Usage.PromptTokens > 0 {
	l.lastPromptTokens = resp.Usage.PromptTokens
}
```

```go
func (l *ReActLoop) applyCompression(ctx context.Context, session *Session, messages []llm.Message) []llm.Message {
	if l.compressor == nil {
		return messages
	}
	est := l.lastPromptTokens
	if est <= 0 {
		est = prompt.EstimateTokens(session.Messages)
	}
	if !l.compressor.ShouldCompress(est, len(session.Messages)) {
		return messages
	}
	before := len(session.Messages)
	out, did, err := l.compressor.Compress(ctx, session.Messages)
	if err != nil || !did {
		return messages
	}
	session.Messages = out
	l.emit("context_compressed", map[string]any{
		"before_msgs": before, "after_msgs": len(out),
		"estimated_tokens_before": est,
		"summary_chars": len(l.compressor.PreviousSummary()),
	})
	return session.LLMMessages()
}
```

Note: `RunTurn` appends the user message before the loop; compression must keep that latest user in the tail (protect_last_n). With default protect_last_n=20 this is fine; in the tiny test ProtectLastN=2 still keeps recent msgs.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/prompt/ ./internal/runtime/ -count=1
```

- [ ] **Step 5: Commit** (if requested)

---

### Task 6: App wiring + RebuildGateway fixes Agent

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/agent/agent.go` (optional `SetCompressor` passthrough)

- [ ] **Step 1: Add Agent passthrough**

```go
func (a *Agent) SetCompressor(c *prompt.Compressor) {
	if a != nil && a.Loop != nil {
		a.Loop.SetCompressor(c)
	}
}

func (a *Agent) SetGateway(g *llm.Gateway) {
	a.Gateway = g
	if a.Loop != nil {
		a.Loop.SetGateway(g)
	}
}
```

- [ ] **Step 2: In `App.New` / after Agent creation, build compressor**

```go
func (a *App) wireCompressor() {
	cfg := a.Config.EffectiveCompression()
	if !cfg.Enabled {
		if a.Agent != nil {
			a.Agent.SetCompressor(nil)
		}
		if a.Loop != nil {
			a.Loop.SetCompressor(nil)
		}
		return
	}
	aux := a.Config.EffectiveAuxiliaryCompression()
	var sum prompt.Summarizer
	p, err := llm.BuildProviderFromLLMFields(aux.Provider, aux.TokenKey, aux.Model, nil, "", aux.BaseURL)
	if err == nil {
		sum = &prompt.ProviderSummarizer{Provider: p}
	} else {
		fmt.Fprintf(os.Stderr, "警告: 压缩摘要模型未就绪: %v（压缩将在无摘要时跳过）\n", err)
	}
	comp := prompt.NewCompressor(cfg, sum)
	if a.Agent != nil {
		a.Agent.SetCompressor(comp)
	}
	if a.Loop != nil {
		a.Loop.SetCompressor(comp)
	}
}
```

Call `wireCompressor()` at end of `New` and at end of `RebuildGateway`.

In `RebuildGateway`, after updating gateway:

```go
if a.Agent != nil {
	a.Agent.SetGateway(a.Gateway)
}
if a.Loop != nil {
	a.Loop.SetGateway(a.Gateway)
}
a.wireCompressor()
```

- [ ] **Step 3: Build**

```bash
go test ./internal/app/ ./internal/agent/ ./internal/prompt/ ./internal/runtime/ ./internal/config/ -count=1
go build -o NUL ./cmd/geegoo/ ./cmd/agent-runtime/
```

Expected: PASS / build OK.

- [ ] **Step 4: Commit** (if requested)

---

### Task 7: Docs

**Files:**
- Modify: `docs/architecture/overview.md` — note `internal/prompt` compressor in Agent diagram
- Modify: `docs/architecture/layers/L3-memory/compaction.md` — replace L1–L4 draft with pointer to spec + implemented Hermes-style single compressor
- Modify: `deploy/hermes-parity-comparison.md` — 上下文压缩 → ✅
- Modify: `docs/superpowers/specs/2026-07-12-context-compression-design.md` — status → 已实现（after code lands）

- [ ] **Step 1: Update the three docs**
- [ ] **Step 2: Commit** (if requested)

---

## Spec coverage checklist

| Spec item | Task |
|---|---|
| Config compression + auxiliary | T1 |
| Token threshold trigger | T2 |
| Phase1 clear tools | T3 |
| Phase2 boundaries + align | T3 |
| Phase3 aux LLM summary | T4 |
| Phase4 assemble + sanitize | T3/T4 |
| Skip on summary failure | T4 |
| previous_summary update | T4 |
| System message unchanged | T4 test |
| ReAct hook + lastPromptTokens | T5 |
| Write-back session.Messages | T5 |
| Progress event | T5 |
| App/Agent wiring | T6 |
| Docs / comparison | T7 |
| No Gateway 85% / Anthropic cache / plugins | (out of scope) |

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-07-12-context-compression.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — implement tasks in this session with checkpoints  

Which approach?
