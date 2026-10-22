package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestSignalDiagnoseComposeProviderUsesChatNotSynthesis(t *testing.T) {
	chatProv := &llm.MockProvider{ModelName: "chat-model"}
	synthProv := &llm.MockProvider{ModelName: "synth-model"}
	chatGW := llm.NewGateway(chatProv, llm.GatewayConfig{MaxRetries: 1})
	synthGW := llm.NewGateway(synthProv, llm.GatewayConfig{MaxRetries: 1})
	cfg := &config.AppConfig{LLM: config.LLMConfig{Provider: "openai", Model: "chat-model", TokenKey: "sk-test"}}
	a := &App{Config: cfg, Gateway: chatGW, SynthesisGateway: synthGW}
	a.invalidateComposeProviderCache()
	a.Agent = agent.New(chatGW, runtime.NewExecutor(tools.NewRegistry()), tools.NewRegistry())
	b := a.ResolveSignalDiagnoseComposeBundle()
	if b.Provider == nil {
		t.Fatal("nil provider")
	}
	if b.Source != ComposeSourceGatewayFallback {
		t.Fatalf("source=%q detail=%q", b.Source, b.Detail)
	}
}
