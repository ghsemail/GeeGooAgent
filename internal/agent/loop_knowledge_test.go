package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunTurnExpandsKnowledgeToolOnSkillMatch(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "knowledge-base")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: knowledge-base\ndescription: 知识库、策略文档、查库、WeKnora、文档里怎么写、MACD 策略\n---\n\n先 search_knowledge。\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := procedural.NewLoader(dir)

	registry := tools.NewRegistry()
	called := false
	registry.Register(tools.Tool{
		Name:        "search_knowledge",
		Description: "search kb",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			called = true
			return tools.Result{Status: tools.StatusOK, Summary: "hit MACD pdf"}
		},
	})

	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "k1", Name: "search_knowledge", Arguments: map[string]any{"query": "MACD"}},
				},
				Usage: llm.TokenUsage{Model: "mock"},
			},
			{Content: "按文档：4小时 MACD。", Usage: llm.TokenUsage{Model: "mock"}},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetSkillLoader(loader, 2)
	loop.SetSkillToolExpander(func(names []string) []llm.ToolSchema {
		for _, n := range names {
			if n == "knowledge-base" {
				return registry.Schemas([]string{"search_knowledge"})
			}
		}
		return nil
	})

	result := loop.RunTurn(
		context.Background(),
		runtime.NewSession(),
		"按知识库里的 4 小时 MACD 策略说明一下",
		tools.Context{},
		nil,
	)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if !schemaHas(provider.LastTools, "search_knowledge") {
		t.Fatalf("expected search_knowledge in schemas, got %#v", namesOf(provider.LastTools))
	}
	if !called {
		t.Fatal("expected search_knowledge tool call")
	}
}

func TestRunTurnOmitsKnowledgeToolWithoutSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "knowledge-base")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: knowledge-base\ndescription: 知识库、策略文档、查库、WeKnora、文档里怎么写、MACD 策略\n---\n\n先 search_knowledge。\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := procedural.NewLoader(dir)
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "search_knowledge",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			t.Fatal("search_knowledge should not run")
			return tools.Result{}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "腾讯约 380 港元", Usage: llm.TokenUsage{Model: "mock"}}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetSkillLoader(loader, 2)
	loop.SetSkillToolExpander(func(names []string) []llm.ToolSchema {
		for _, n := range names {
			if n == "knowledge-base" {
				return registry.Schemas([]string{"search_knowledge"})
			}
		}
		return nil
	})
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "腾讯现在股价多少", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if schemaHas(provider.LastTools, "search_knowledge") {
		t.Fatal("search_knowledge must not appear in default turn schemas")
	}
}

func schemaHas(schemas []llm.ToolSchema, name string) bool {
	for _, s := range schemas {
		if s.Name == name {
			return true
		}
	}
	return false
}

func namesOf(schemas []llm.ToolSchema) []string {
	out := make([]string, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, s.Name)
	}
	return out
}
