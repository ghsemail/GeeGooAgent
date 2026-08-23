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
	useOps := false
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:    "openai",
				TokenKey:    "test-key",
				Model:       "test-model",
				UseOpsModel: &useOps,
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
}

func boolPtr(v bool) *bool {
	return &v
}
