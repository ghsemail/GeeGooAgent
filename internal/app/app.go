package app

import (
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// App wires config, MCP client, tools, LLM, and workflow.
type App struct {
	Config    *config.AppConfig
	MCP       *mcp.Client
	Registry  *tools.Registry
	Gateway   *llm.Gateway
	Loop      *runtime.ReActLoop
	Executor  *runtime.Executor
	Workflow  *workflow.Runner
	Working   *memory.WorkingStore
	State     *infra.StateStore
	EventBus  *infra.EventBus
	Workspace string
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

	workspace, err := cfg.ResolveOutputDir()
	if err != nil {
		return nil, err
	}
	state := infra.NewStateStore(workspace)
	working := memory.NewWorkingStore(state)
	checkpoints := infra.NewCheckpointManager(state)
	eventBus := infra.NewEventBus()

	mcpClient := mcp.NewClient(cfg.EffectiveMCPURL(), cfg.MCPAPIKey(), mcp.Options{
		AllowedHosts: cfg.ResolvedAllowedHosts(),
	})

	registry := tools.NewRegistry()
	workingLoader := workflow.WorkingLoaderAdapter{Store: working}
	tools.RegisterAll(registry, tools.Deps{
		MCP: mcpClient, WorkspaceRoot: workspace, ProjectRoot: findProjectRoot(),
		Working: workingLoader,
	})

	executor := runtime.NewExecutor(registry)
	cpAdapter := workflow.CheckpointAdapter{SaveFn: func(sessionID, skill, status, lastTool string, step int, w *memory.PreMarketWorking) error {
		return checkpoints.Save(infra.Checkpoint{
			SessionID: sessionID, Step: step, Skill: skill, Status: status, LastTool: lastTool,
			Working: encodeWorkingMap(w),
		})
	}}
	wf := workflow.NewRunner(executor, working, cpAdapter)

	var gateway *llm.Gateway
	provider, err := llm.BuildProviderFromConfig(cfg.LLM.Provider, cfg.LLM.TokenKey, cfg.LLM.Model)
	if err == nil {
		gateway = llm.NewGateway(provider, llm.GatewayConfig{
			MaxRetries: 3, Temperature: cfg.LLM.Temperature, MaxTokens: cfg.LLM.MaxTokens,
		})
	}

	return &App{
		Config: cfg, MCP: mcpClient, Registry: registry, Gateway: gateway,
		Loop: runtime.NewReActLoop(gateway, executor), Executor: executor,
		Workflow: wf, Working: working, State: state, EventBus: eventBus, Workspace: workspace,
	}, nil
}

// RunPreMarket executes the pre_market skill workflow.
func (a *App) RunPreMarket(skill string) (workflow.RunResult, error) {
	if skill != "pre_market" {
		return workflow.RunResult{}, fmt.Errorf("unsupported skill: %s", skill)
	}
	sessionID := newSessionID()
	a.EventBus.Emit("RunStarted", map[string]any{"session_id": sessionID, "skill": skill})
	working, err := a.Working.Create(sessionID, skill)
	if err != nil {
		return workflow.RunResult{}, err
	}
	ctx := a.ToolContext(sessionID)
	result := a.Workflow.Run(sessionID, skill, workflow.PhaseASteps(), workflow.PerStockSteps(), ctx, working)
	return result, nil
}

// ToolContext builds execution context for the current session.
func (a *App) ToolContext(sessionID string) tools.Context {
	return tools.Context{
		SessionID: sessionID, MCPToken: a.Config.MCPToken(), DryRun: a.Config.DryRun,
		WorkspaceRoot: a.Workspace, EventBus: a.EventBus, StateStore: a.State,
	}
}

// EndpointSummary prints GeeGoo service endpoints.
func (a *App) EndpointSummary() string {
	return fmt.Sprintf(
		"GeeGooBot mcp-api %s | GeeGooSignal catalog %s | GeeGooData %s",
		a.Config.EffectiveMCPURL(), a.Config.SignalCatalogURL(), a.Config.DataHTTPURL(),
	)
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func newSessionID() string {
	return "run-" + fmt.Sprintf("%d", os.Getpid())
}

func encodeWorkingMap(w *memory.PreMarketWorking) map[string]any {
	stocks := map[string]any{}
	for k, v := range w.Stocks {
		stocks[k] = map[string]any{"status": v.Status, "code": v.Code}
	}
	out := map[string]any{
		"session_id": w.SessionID, "skill": w.Skill, "phase": w.Phase, "stocks": stocks,
	}
	if w.IsTradingDay != nil {
		out["is_trading_day"] = *w.IsTradingDay
	}
	return out
}
