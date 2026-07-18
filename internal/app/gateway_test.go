package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRebuildGatewayUpdatesAgentGateway(t *testing.T) {
	registry := tools.NewRegistry()
	initial := llm.NewGateway(&llm.MockProvider{}, llm.GatewayConfig{MaxRetries: 1})
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider: "openai",
				TokenKey: "test-key",
				Model:    "test-model",
			},
			Compression: config.CompressionConfig{Enabled: boolPtr(false)},
		},
		Registry: registry,
		Gateway:  initial,
		Agent:    agent.New(initial, runtime.NewExecutor(registry), registry),
	}

	if err := application.RebuildGateway(); err != nil {
		t.Fatal(err)
	}
	if application.Agent.Gateway != application.Gateway {
		t.Fatal("agent gateway was not synchronized")
	}
	if application.Agent.ReportSynthesizer() == nil {
		t.Fatal("report synthesizer not wired")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
