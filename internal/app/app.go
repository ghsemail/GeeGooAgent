package app

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/skills"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

var fallbackSessionCounter uint64

// App wires config, MCP client, tools, LLM, and workflow.
type App struct {
	Config      *config.AppConfig
	MCP         *mcp.Client
	Registry    *tools.Registry
	Gateway     *llm.Gateway
	Loop        *runtime.ReActLoop
	Executor    *runtime.Executor
	Workflow    *workflow.Runner
	Working     *memory.WorkingStore
	State       *infra.StateStore
	Checkpoints *infra.CheckpointManager
	EventBus    *infra.EventBus
	Workspace   string
	// P1 SQLite foundation. DB is nil when disabled via GEEGOO_DB=off or open failure.
	DB       *infra.DB
	Evidence *memory.EvidenceStore
	// P2c platform-agnostic agent core. Owns the ReAct loop; used by chat,
	// runtime HTTP, and (later) workflow/scheduler.
	Agent *agent.Agent
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
		Working: workingLoader, Search: cfg.EffectiveSearch(),
	})

	executor := runtime.NewExecutor(registry)
	cpAdapter := workflow.CheckpointAdapter{SaveFn: func(sessionID, skill, status, lastTool string, step int, w *memory.PreMarketWorking) error {
		return checkpoints.Save(infra.Checkpoint{
			SessionID: sessionID, Step: step, Skill: skill, Status: status, LastTool: lastTool,
			Working: encodeWorkingMap(w),
		})
	}}
	wf := workflow.NewRunner(executor, working, cpAdapter)

	app := &App{
		Config: cfg, MCP: mcpClient, Registry: registry,
		Executor: executor, Workflow: wf, Working: working, State: state, Checkpoints: checkpoints, EventBus: eventBus, Workspace: workspace,
	}
	if err := app.openDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: SQLite 未启用: %v（回退到文件存储）\n", err)
	}
	if err := app.RebuildGateway(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: LLM 未就绪: %v\n", err)
	}
	app.Loop = runtime.NewReActLoop(app.Gateway, executor)
	app.Agent = agent.New(app.Gateway, executor, registry)
	app.wireCompressor()
	if adapter := newSynthesizerAdapter(app.Gateway, app.Config.LLM.Model); adapter != nil {
		workflow.SetDefaultSynthesizer(adapter)
	}

	return app, nil
}

// openDatabase opens the SQLite store at workspace/geegoo.db unless
// GEEGOO_DB=off. On success wires EvidenceStore. Failure is non-fatal:
// callers fall back to the legacy file StateStore.
func (a *App) openDatabase() error {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_DB"))); v == "off" || v == "0" || v == "false" {
		return nil
	}
	dbPath := filepath.Join(a.Workspace, "geegoo.db")
	db, err := infra.OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	a.DB = db
	a.Evidence = memory.NewEvidenceStore(db)
	return nil
}

// Close releases resources owned by the App (currently the SQLite handle).
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

// RebuildGateway recreates the LLM gateway from current config (after /think or /model).
// When llm.use_ops_model is true/nil, prefers ops configured model from Signal catalog/admin.
func (a *App) RebuildGateway() error {
	providerName := a.Config.LLM.Provider
	tokenKey := a.Config.LLM.TokenKey
	model := a.Config.LLM.Model
	baseURL := strings.TrimSpace(a.Config.LLM.BaseURL)

	if a.Config.LLM.OpsModelEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		doc, src, err := admin.QueryConfiguredFromCandidates(ctx, a.Config.AdminModelURLs()...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 拉取运营配置模型失败（回退本地 llm）: %v\n", err)
		} else {
			name := strings.TrimSpace(doc.Name)
			if name == "" {
				name = strings.TrimSpace(doc.DisplayName)
			}
			if tok := strings.TrimSpace(doc.Token); tok != "" {
				tokenKey = tok
			}
			if name != "" {
				model = name
			}
			if bu := strings.TrimSpace(doc.BaseURL); bu != "" {
				baseURL = bu
			}
			providerName = llm.InferProviderFromNames(doc.DisplayName, doc.Name)
			fmt.Fprintf(os.Stderr, "LLM: 使用运营配置 model=%s base_url=%s from %s\n", model, baseURL, src)
		}
	}

	provider, err := llm.BuildProviderFromLLMFields(
		providerName, tokenKey, model,
		a.Config.LLM.Thinking, a.Config.LLM.ReasoningEffort, baseURL,
	)
	if err != nil {
		return err
	}
	a.Gateway = llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries: 3, Temperature: a.Config.LLM.Temperature, MaxTokens: a.Config.LLM.MaxTokens,
	})
	if a.Loop != nil {
		a.Loop.SetGateway(a.Gateway)
	}
	if a.Agent != nil {
		a.Agent.SetGateway(a.Gateway)
	}
	a.wireCompressor()
	return nil
}

func (a *App) wireCompressor() {
	cfg := a.Config.EffectiveCompression()
	if !cfg.Enabled {
		a.setCompressor(nil)
		return
	}
	aux := a.Config.EffectiveAuxiliaryCompression()
	provider, err := llm.BuildProviderFromLLMFields(aux.Provider, aux.TokenKey, aux.Model, nil, "", aux.BaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 上下文压缩未启用: %v\n", err)
		a.setCompressor(nil)
		return
	}
	a.setCompressor(prompt.NewCompressor(cfg, &prompt.ProviderSummarizer{Provider: provider}))
}

func (a *App) setCompressor(c *prompt.Compressor) {
	if a.Agent != nil {
		a.Agent.SetCompressor(c)
	}
	if a.Loop != nil {
		a.Loop.SetCompressor(c)
	}
}

// Skills is the registry of runnable skills (built-in + any registered at runtime).
var DefaultSkills = skills.Default()

// RunPreMarket executes the pre_market skill workflow.
// Kept for backward compatibility; new callers should use RunSkill.
func (a *App) RunPreMarket(skill string) (workflow.RunResult, error) {
	return a.RunSkill(skill)
}

// RunSkill executes a named skill workflow looked up in the skill registry.
// Returns an error if the skill is not registered.
func (a *App) RunSkill(skill string) (workflow.RunResult, error) {
	return a.RunSkillContext(context.Background(), skill)
}

// RunSkillContext executes a named skill with cancellation propagated to tools
// and optional LLM synthesis.
func (a *App) RunSkillContext(ctx context.Context, skill string) (workflow.RunResult, error) {
	spec, ok := DefaultSkills.Get(skill)
	if !ok {
		return workflow.RunResult{}, fmt.Errorf("unknown skill: %s (run 'geegoo skills list')", skill)
	}
	if spec.PhaseA == nil || spec.PerStock == nil {
		return workflow.RunResult{}, fmt.Errorf("skill %s has no step functions defined", skill)
	}
	phaseA := spec.PhaseA()
	perStock := spec.PerStock()
	if len(phaseA) == 0 && len(perStock) == 0 {
		return workflow.RunResult{}, fmt.Errorf("skill %s is registered but has no executable steps", skill)
	}
	sessionID := newSessionID()
	a.EventBus.Emit("RunStarted", map[string]any{"session_id": sessionID, "skill": skill})
	working, err := a.Working.Create(sessionID, skill)
	if err != nil {
		return workflow.RunResult{}, err
	}
	toolCtx := a.ToolContextWithContext(ctx, sessionID)
	result := a.Workflow.Run(sessionID, skill, phaseA, perStock, toolCtx, working)
	return result, nil
}

// ResumePreMarket resumes a workflow from its latest checkpoint. The checkpoint's
// skill name drives step lookup via the registry, so resume works for any skill.
func (a *App) ResumePreMarket(sessionID string) (workflow.RunResult, error) {
	return a.ResumePreMarketContext(context.Background(), sessionID)
}

// ResumePreMarketContext resumes a workflow and propagates cancellation to tool calls.
func (a *App) ResumePreMarketContext(ctx context.Context, sessionID string) (workflow.RunResult, error) {
	cp, err := a.Checkpoints.LoadLatest(sessionID)
	if err != nil {
		return workflow.RunResult{}, err
	}
	if cp == nil {
		return workflow.RunResult{}, fmt.Errorf("checkpoint not found for session: %s", sessionID)
	}
	spec, ok := DefaultSkills.Get(cp.Skill)
	if !ok || spec.PhaseA == nil || spec.PerStock == nil {
		return workflow.RunResult{}, fmt.Errorf("unsupported checkpoint skill: %s", cp.Skill)
	}
	phaseA := spec.PhaseA()
	perStock := spec.PerStock()
	if len(phaseA) == 0 && len(perStock) == 0 {
		return workflow.RunResult{}, fmt.Errorf("checkpoint skill %s has no executable steps", cp.Skill)
	}
	working, err := a.Working.Load(sessionID)
	if err != nil {
		return workflow.RunResult{}, err
	}
	if working == nil {
		return workflow.RunResult{}, fmt.Errorf("working state not found for session: %s", sessionID)
	}
	if cp.Status == "completed" || working.Phase == "done" {
		return workflow.RunResult{SessionID: sessionID, Status: "completed", Working: working}, nil
	}
	toolCtx := a.ToolContextWithContext(ctx, sessionID)
	return a.Workflow.RunFrom(sessionID, cp.Skill, phaseA, perStock, toolCtx, working, cp.Step), nil
}

// ToolContext builds execution context for the current session.
func (a *App) ToolContext(sessionID string) tools.Context {
	return a.ToolContextWithContext(context.Background(), sessionID)
}

// ToolContextWithContext builds execution context for the current session.
func (a *App) ToolContextWithContext(ctx context.Context, sessionID string) tools.Context {
	return tools.Context{
		Ctx: ctx, SessionID: sessionID, MCPToken: a.Config.MCPToken(), DryRun: a.Config.DryRun,
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
	var suffix [4]byte
	if _, err := cryptorand.Read(suffix[:]); err == nil {
		return fmt.Sprintf("run-%s-%d-%s", time.Now().UTC().Format("20060102T150405.000000000Z"), os.Getpid(), hex.EncodeToString(suffix[:]))
	}
	return fmt.Sprintf("run-%s-%d-%d", time.Now().UTC().Format("20060102T150405.000000000Z"), os.Getpid(), atomic.AddUint64(&fallbackSessionCounter, 1))
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
