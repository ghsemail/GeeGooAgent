package app

import (
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// App wires config, MCP client, tools, and LLM gateway.
type App struct {
	Config   *config.AppConfig
	MCP      *mcp.Client
	Registry *tools.Registry
	Gateway  *llm.Gateway
	Loop     *runtime.ReActLoop
}

// LoadFromConfigPath builds an App from config.json.
func LoadFromConfigPath(path string, dryRun bool) (*App, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if dryRun {
		cfg.DryRun = true
	}
	for _, w := range cfg.LegacyPortWarnings() {
		fmt.Fprintf(os.Stderr, "警告: %s\n", w)
	}

	mcpClient := mcp.NewClient(cfg.EffectiveMCPURL(), cfg.MCPAPIKey(), mcp.Options{
		AllowedHosts: cfg.ResolvedAllowedHosts(),
	})

	registry := tools.NewRegistry()
	tools.RegisterChatMCPTools(registry, tools.MCPDeps{Client: mcpClient})

	provider, err := llm.BuildProviderFromConfig(cfg.LLM.Provider, cfg.LLM.TokenKey, cfg.LLM.Model)
	if err != nil {
		return nil, err
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries:  3,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	})
	executor := runtime.NewExecutor(registry)
	loop := runtime.NewReActLoop(gateway, executor)

	return &App{
		Config:   cfg,
		MCP:      mcpClient,
		Registry: registry,
		Gateway:  gateway,
		Loop:     loop,
	}, nil
}

// ToolContext builds execution context for the current session.
func (a *App) ToolContext(sessionID string) tools.Context {
	return tools.Context{
		SessionID: sessionID,
		MCPToken:  a.Config.MCPToken(),
		DryRun:    a.Config.DryRun,
	}
}

// EndpointSummary prints GeeGoo service endpoints (not Trading legacy).
func (a *App) EndpointSummary() string {
	return fmt.Sprintf(
		"GeeGooBot mcp-api %s | GeeGooSignal catalog %s | GeeGooData %s",
		a.Config.EffectiveMCPURL(),
		a.Config.SignalCatalogURL(),
		a.Config.DataHTTPURL(),
	)
}
