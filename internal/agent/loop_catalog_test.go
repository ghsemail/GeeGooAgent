package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/agent/decision"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestCatalogModeDoesNotKeywordInjectFullSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "strategy-backtest-run")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: strategy-backtest-run\ndescription: 回测、跑回测、SAR MACD\n---\n\nSECRET_FULL_PLAYBOOK_BODY\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &llm.MockProvider{Responses: []*llm.Response{{Content: "ok"}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetSkillLoader(procedural.NewLoader(dir), 4)
	loop.SetSkillInjection(procedural.SkillInjectionCatalog)

	var injectSource string
	var sawSecret bool
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "memory.procedural" {
			if s, ok := data["source"].(string); ok {
				injectSource = s
			}
		}
	})

	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "帮我回测小米 MACD", tools.Context{}, nil)

	for _, rec := range session.LLMMessages() {
		if strings.Contains(rec.Content, "SECRET_FULL_PLAYBOOK_BODY") {
			sawSecret = true
		}
	}
	if sawSecret {
		t.Fatal("catalog mode must not inject full playbook body via keyword match")
	}
	if injectSource != "catalog" {
		t.Fatalf("expected catalog inject source, got %q", injectSource)
	}
}

func TestCapabilityRouteShadowEmitsDecision(t *testing.T) {
	root := findProjectSkillsRoot(t)
	loader := procedural.NewLoader(root)
	if n := len(loader.ChatCatalog()); n == 0 {
		t.Fatalf("expected non-empty catalog at %s", root)
	}
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: `{"choice_id":"_none","confidence":0.5,"reason":"test"}`},
			{Content: "你好，有什么可以帮你？"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetSkillLoader(loader, 2)
	loop.SetSkillInjection(procedural.SkillInjectionCatalog)
	loop.SetDecisionCoordinator(&decision.Coordinator{
		Mode: decision.ModeShadow,
		Hook: decision.StubHook{ChoiceID: "_none", Confidence: 0.5},
	})

	var sawCapability bool
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "decision" && data["kind"] == string(decision.KindCapabilityRoute) {
			sawCapability = true
		}
	})

	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "你好", tools.Context{}, nil)
	if !sawCapability {
		t.Fatal("expected capability_route decision SSE in shadow mode")
	}
}

func findProjectSkillsRoot(t *testing.T) string {
	t.Helper()
	for _, rel := range []string{
		filepath.Join("..", "..", "skills"),
		filepath.Join("skills"),
	} {
		if st, err := os.Stat(rel); err == nil && st.IsDir() {
			if _, err := os.Stat(filepath.Join(rel, "signal_diagnose", "SKILL.md")); err == nil {
				abs, _ := filepath.Abs(rel)
				return abs
			}
		}
	}
	t.Fatal("skills root not found")
	return ""
}
