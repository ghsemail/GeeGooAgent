package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type mockComposeLLM struct {
	content string
}

func (m *mockComposeLLM) Model() string { return "mock" }

func (m *mockComposeLLM) Chat(_ context.Context, _ []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	return &llm.Response{Content: m.content}, nil
}

func TestSynthesizeCognitionBodyUsesLLM(t *testing.T) {
	runner := &Runner{
		ComposeLLM: &mockComposeLLM{
			content: "## 一句话定位\n\n测试策略\n\n## 适用场景\n\n震荡市",
		},
	}
	flow := &Flow{
		CatalogLabel: "Macd4H",
		CatalogType:  catalogTypeCombination,
		CatalogRaw:   map[string]any{"name": "Macd4H", "signal_id": "abc"},
	}
	body, err := runner.synthesizeCognitionBody(context.Background(), flow)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if !strings.Contains(body, "测试策略") {
		t.Fatalf("body=%q", body)
	}
	doc := assembleAgentCognitionDoc(flow, body)
	if !strings.Contains(doc, "agent_use:") || !strings.Contains(doc, "适用场景") {
		t.Fatalf("assembled doc missing frontmatter/sections: %s", doc)
	}
	if !strings.Contains(doc, "doc_type: strategy_agent_archive") {
		t.Fatalf("assembled doc should follow template.md: %s", doc)
	}
}

func TestArchiveTemplateHeadingsDriveCompose(t *testing.T) {
	tpl := loadStrategyArchiveTemplate()
	if tpl == "" {
		t.Fatal("expected skills/generate_strategy_archive/template.md")
	}
	headings := archiveRequiredHeadings(tpl)
	if len(headings) < 6 {
		t.Fatalf("headings=%v", headings)
	}
	prompt := cognitionComposeSystemPrompt()
	for _, h := range headings {
		if !strings.Contains(prompt, "## "+h) {
			t.Fatalf("system prompt missing heading %q", h)
		}
	}
}
