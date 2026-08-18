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
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/memory/consolidation"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memory/semantic"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/skills"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/userllmstore"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

var fallbackSessionCounter uint64

// App wires config, MCP client, tools, LLM, and workflow.
type App struct {
	Config      *config.AppConfig
	MCP         *mcp.Client
	Registry    *tools.Registry
	Gateway     *llm.Gateway
	Executor    *runtime.Executor
	Workflow    *workflow.Runner
	Working     *memory.WorkingStore
	State       *infra.StateStore
	Checkpoints *infra.CheckpointManager
	EventBus    *infra.EventBus
	Workspace   string
	// P1 SQLite foundation. DB is nil when disabled via GEEGOO_DB=off or open failure.
	DB       *infra.DB
	// Optional PostgreSQL platform DB (sessions, cockpit, semantic memory).
	PG *infra.PostgresDB
	// Semantic memory chunks when pgvector schema is enabled (legacy opt-in; Waku semantic = agent_facts).
	Semantic *semantic.PostgresStore
	// Facts is Waku-style semantic memory (FTS facts table).
	Facts *facts.PostgresStore
	// Episodic memory (dated summaries) when PostgreSQL is enabled.
	Episodic *episodic.PostgresStore
	// Consolidator distills chats into semantic facts + episodic rows.
	Consolidator *consolidation.Distiller
	// Procedural memory scans SKILL.md under skills/.
	SkillLoader *procedural.Loader
	Evidence *memory.EvidenceStore
	// P2c platform-agnostic agent core. Owns the ReAct loop; used by chat,
	// runtime HTTP, and (later) workflow/scheduler.
	Agent *agent.Agent
	Hooks *tools.HookRunner
	// ChatMemory is the Memory port shared by loop, recall tool, and evidence.
	ChatMemory memport.Port
	// UserLLM loads per-user gateway model prefs from GeeGooBot DB (service-api or Mongo).
	UserLLM *userllmstore.Backend
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

	mcpOpts := mcp.Options{AllowedHosts: cfg.ResolvedAllowedHosts()}
	mcpLimiter := mcp.NewConcurrencyLimiter(cfg.EffectiveMCPMaxParallel())
	mcpOpts.Concurrency = mcpLimiter
	analysisOpts := mcpOpts
	analysisOpts.Timeout = 5 * time.Minute
	httpBackends := tools.HTTPBackends{
		MCP:           mcp.NewClient(cfg.EffectiveMCPURL(), cfg.MCPAPIKey(), analysisOpts),
		SignalAPI:     mcp.NewClient(cfg.SignalAPIURL(), cfg.SignalAPIKey(), mcpOpts),
		SignalCatalog: mcp.NewClient(cfg.SignalCatalogURL(), cfg.SignalCatalogAPIKey(), mcpOpts),
		SignalAnalyze: mcp.NewClient(cfg.SignalAnalyzeURL(), cfg.SignalAnalyzeAPIKey(), analysisOpts),
	}

	registry := tools.NewRegistry()
	workingLoader := workflow.WorkingLoaderAdapter{Store: working}
	executor := runtime.NewExecutor(registry)
	cpAdapter := workflow.CheckpointAdapter{SaveFn: func(sessionID, skill, status, lastTool string, step int, w *memory.PreMarketWorking) error {
		return checkpoints.Save(infra.Checkpoint{
			SessionID: sessionID, Step: step, Skill: skill, Status: status, LastTool: lastTool,
			Working: encodeWorkingMap(w),
		})
	}}
	wf := workflow.NewRunner(executor, working, cpAdapter)

	app := &App{
		Config: cfg, MCP: httpBackends.MCP, Registry: registry,
		Executor: executor, Workflow: wf, Working: working, State: state, Checkpoints: checkpoints, EventBus: eventBus, Workspace: workspace,
		UserLLM: userllmstore.NewBackend(cfg, workspace),
	}
	if err := app.openDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: SQLite 未启用: %v（回退到文件存储）\n", err)
	}
	if err := app.openPostgres(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: PostgreSQL 未连接: %v\n", err)
	}
	if err := app.RebuildGateway(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: LLM 未就绪: %v\n", err)
	}
	app.Hooks = buildHooks(cfg.Hooks)
	app.Agent = agent.New(app.Gateway, executor, registry)
	app.Agent.SetMaxToolRounds(cfg.EffectiveMaxSteps())
	app.Agent.SetToolMaxParallel(cfg.EffectiveToolMaxParallel())
	app.Agent.SetToolTimeout(cfg.EffectiveToolTimeout())
	app.Agent.SetPlanGate(cfg.EffectivePlanGate())
	app.Agent.SetEvalMaxRetries(cfg.EffectiveEvalMaxRetries())
	app.Agent.SetDelegateMaxParallel(cfg.EffectiveDelegateMaxParallel())
	app.Agent.SetEventBus(eventBus)
	sub := agent.NewSubAgent(agent.SubAgentConfig{
		Gateway: app.Gateway, Executor: executor, Registry: registry,
		MaxSteps: cfg.EffectiveSubAgentMaxSteps(),
		MaxParallel: cfg.EffectiveDelegateMaxParallel(),
		ChatToolNames: app.ChatToolNames,
	})
	sub.SetEventBus(eventBus)
	app.wireChatMemory()
	app.wireCognition()
	app.wireRecallRanker()
	tools.RegisterAll(registry, tools.Deps{
		HTTP: httpBackends, WorkspaceRoot: workspace, ProjectRoot: findProjectRoot(),
		Working: workingLoader, Search: cfg.EffectiveSearch(),
		FeishuWebhookURL: cfg.EffectiveFeishuWebhookURL(),
		Delegate: sub, Memory: app.ChatMemory,
		Facts: app.Facts, Episodic: app.Episodic, Home: config.Home(),
		SkillLoader: app.SkillLoader,
	})
	app.Agent.SetSubAgent(sub)
	app.Workflow.SetToolExec(app.Agent.ToolExec())
	app.wireSynthesizer()

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
	var err error
	if a.PG != nil {
		if closeErr := a.PG.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if a.DB != nil {
		if closeErr := a.DB.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

func (a *App) openPostgres() error {
	dsn := infra.PostgresDSN()
	if dsn == "" {
		return nil
	}
	pg, err := infra.OpenPostgres(dsn)
	if err != nil {
		return err
	}
	a.PG = pg
	a.Episodic = episodic.NewPostgresStore(pg.SQL())
	a.Facts = facts.NewPostgresStore(pg.SQL())
	if n, _ := a.Facts.Count(context.Background(), ""); n == 0 {
		if imported, err := a.Facts.MigrateFromLegacyChunks(context.Background()); err == nil && imported > 0 {
			fmt.Fprintf(os.Stderr, "已迁移 %d 条 legacy semantic facts\n", imported)
		}
	}
	if legacySessionVectorsEnabled() && vectorEnabled() {
		dim := config.DefaultEmbeddingDimensions
		if a.Config != nil {
			dim = a.Config.ResolvedEmbedding().Dimensions
		}
		if err := pg.ApplyMemorySchema(dim); err != nil {
			fmt.Fprintf(os.Stderr, "警告: legacy session vector schema 未应用 (%v)\n", err)
		} else {
			a.Semantic = semantic.NewPostgresStore(pg.SQL(), a.Config)
			fmt.Fprintf(os.Stderr, "提示: legacy session vectors 已启用 (GEEGOO_LEGACY_SESSION_VECTORS=1)；Waku semantic 请用 agent_facts\n")
		}
	}
	return nil
}

func legacySessionVectorsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_LEGACY_SESSION_VECTORS")))
	return v == "1" || v == "true" || v == "yes"
}

func vectorEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_VECTOR_ENABLE")))
	return v == "1" || v == "true" || v == "yes"
}

// RebuildGateway recreates the LLM gateway from current config (after /think or /model).
// When llm.use_ops_model is true/nil, prefers ops configured model from Signal catalog/admin.
func (a *App) RebuildGateway() error {
	if a == nil || a.Config == nil {
		return fmt.Errorf("app not configured")
	}
	gw, resolved, err := a.BuildGatewayFromLLMConfig(a.Config.LLM, true)
	if err != nil {
		return err
	}
	if resolved != nil {
		a.syncLLMConfigFromResolved(resolved.provider, resolved.tokenKey, resolved.model, resolved.baseURL)
	}
	a.Gateway = gw
	if a.Agent != nil {
		a.Agent.SetGateway(a.Gateway)
	}
	a.wireChatMemory()
	a.wireSynthesizer()
	return nil
}

type resolvedLLMFields struct {
	provider, tokenKey, model, baseURL string
}

// BuildGatewayFromLLMConfig builds a gateway from cfg without mutating a.Config unless syncGlobal is true.
func (a *App) BuildGatewayFromLLMConfig(cfg config.LLMConfig, syncGlobal bool) (*llm.Gateway, *resolvedLLMFields, error) {
	if a == nil || a.Config == nil {
		return nil, nil, fmt.Errorf("app not configured")
	}
	providerName := cfg.Provider
	tokenKey := cfg.TokenKey
	model := cfg.Model
	baseURL := strings.TrimSpace(cfg.BaseURL)
	var resolved *resolvedLLMFields

	if cfg.OpsModelEnabled() && strings.TrimSpace(cfg.CatalogModelID) == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		targets := make([]admin.QueryTarget, 0, len(a.Config.AdminModelQueryTargets()))
		for _, t := range a.Config.AdminModelQueryTargets() {
			targets = append(targets, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
		}
		doc, src, err := admin.QueryConfiguredFromTargets(ctx, targets...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 拉取运营配置模型失败（回退本地 llm）: %v\n", err)
		} else {
			a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
			resolved = &resolvedLLMFields{provider: providerName, tokenKey: tokenKey, model: model, baseURL: baseURL}
			if syncGlobal {
				fmt.Fprintf(os.Stderr, "LLM: 使用运营配置 model=%s base_url=%s from %s\n", model, baseURL, src)
			}
		}
	} else if id := strings.TrimSpace(cfg.CatalogModelID); id != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		targets := make([]admin.QueryTarget, 0, len(a.Config.AdminModelQueryTargets()))
		for _, t := range a.Config.AdminModelQueryTargets() {
			targets = append(targets, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
		}
		doc, src, err := admin.QueryModelFromTargets(ctx, targets, id, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 拉取 catalog 模型失败（回退本地 llm）: %v\n", err)
		} else {
			a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
			resolved = &resolvedLLMFields{provider: providerName, tokenKey: tokenKey, model: model, baseURL: baseURL}
			if syncGlobal {
				fmt.Fprintf(os.Stderr, "LLM: 使用 catalog 模型 model=%s base_url=%s from %s\n", model, baseURL, src)
			}
		}
	}

	provider, err := llm.BuildProviderFromLLMFields(
		providerName, tokenKey, model,
		cfg.Thinking, cfg.ReasoningEffort, baseURL,
		cfg.PromptCache,
	)
	if err != nil {
		return nil, nil, err
	}
	thinkingOn := llm.ResolveThinkingEnabled(llm.ProviderName(providerName), model, cfg.Thinking)
	maxTokens := cfg.EffectiveMaxTokens(thinkingOn)
	temp := cfg.Temperature
	gw := llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries: 3, RetryWait: time.Second, Temperature: temp, MaxTokens: maxTokens,
	})
	gw.SetPolicy(a.buildModelPolicy(thinkingOn, maxTokens, temp))
	gw.SetFallbacks(a.buildFallbackProviders())
	return gw, resolved, nil
}

func (a *App) applyCatalogModelDoc(doc *admin.ConfiguredModel, providerName, tokenKey, model, baseURL *string) {
	if doc == nil {
		return
	}
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		name = strings.TrimSpace(doc.DisplayName)
	}
	if tok := strings.TrimSpace(doc.Token); tok != "" {
		*tokenKey = tok
	}
	if name != "" {
		*model = name
	}
	if bu := strings.TrimSpace(doc.BaseURL); bu != "" {
		*baseURL = bu
	}
	if p := strings.TrimSpace(doc.Provider); p != "" {
		*providerName = p
	} else {
		*providerName = llm.InferProviderFromNames(doc.DisplayName, doc.Name)
	}
}

func (a *App) syncLLMConfigFromResolved(providerName, tokenKey, model, baseURL string) {
	if a == nil || a.Config == nil {
		return
	}
	if m := strings.TrimSpace(model); m != "" {
		a.Config.LLM.Model = m
	}
	if p := strings.TrimSpace(providerName); p != "" {
		a.Config.LLM.Provider = p
	}
	if bu := strings.TrimSpace(baseURL); bu != "" {
		a.Config.LLM.BaseURL = bu
	}
	if tok := strings.TrimSpace(tokenKey); tok != "" {
		a.Config.LLM.TokenKey = tok
	}
}

// EffectiveLLMModel returns the active chat model (gateway wins over config).
func (a *App) EffectiveLLMModel() string {
	if a != nil && a.Gateway != nil {
		if m := strings.TrimSpace(a.Gateway.Model()); m != "" {
			return m
		}
	}
	if a == nil || a.Config == nil {
		return ""
	}
	cfg := a.Config.LLM
	return llm.ResolveModel(llm.ProviderName(cfg.Provider), cfg.Model)
}

// EffectiveLLMConfig merges per-user gateway prefs from QT_DB when UserLLM backend is enabled.
func (a *App) EffectiveLLMConfig(userID, gateway string) config.LLMConfig {
	if a == nil || a.Config == nil {
		return config.LLMConfig{}
	}
	base := a.Config.LLM
	userID = strings.TrimSpace(userID)
	if userID == "" || a.UserLLM == nil {
		return base
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	doc, err := a.UserLLM.Load(ctx, userID)
	if err != nil || doc == nil {
		return base
	}
	return userllmstore.MergeEffective(base, doc, gateway)
}

func (a *App) buildFallbackProviders() []llm.Provider {
	if a == nil || a.Config == nil {
		return nil
	}
	var out []llm.Provider
	for _, fb := range a.Config.LLM.Fallbacks {
		if strings.TrimSpace(fb.TokenKey) == "" {
			continue
		}
		p, err := llm.BuildProviderFromLLMFields(
			fb.Provider, fb.TokenKey, fb.Model,
			fb.Thinking, fb.ReasoningEffort, fb.BaseURL,
			fb.PromptCache,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: fallback LLM 跳过 (%s): %v\n", fb.Provider, err)
			continue
		}
		out = append(out, p)
	}
	return out
}

func (a *App) wireChatMemory() {
	if a == nil {
		return
	}
	cfg := a.Config.EffectiveCompression()
	var compressor *prompt.Compressor
	if cfg.Enabled {
		model := ""
		if a.Gateway != nil {
			model = a.Gateway.Model()
		}
		if model == "" && a.Config != nil {
			model = a.Config.LLM.Model
		}
		cfg.ContextLength = llm.ResolveContextWindow(model, a.Config.Compression.ContextLength)
		aux := a.Config.EffectiveAuxiliaryCompression()
		provider, err := llm.BuildProviderFromLLMFields(aux.Provider, aux.TokenKey, aux.Model, nil, "", aux.BaseURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 上下文压缩未启用: %v\n", err)
		} else {
			var policy llm.Policy
			if a.Gateway != nil {
				policy = a.Gateway.Policy()
			}
			compressor = prompt.NewCompressor(cfg, &prompt.ProviderSummarizer{
				Provider: provider,
				Policy:   policy,
			})
		}
	}
	sessions, err := a.SessionStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: memory 会话存储回退 file: %v\n", err)
		if a.State != nil {
			sessions = chatsession.NewChatSessionStore(a.State)
		}
	}
	var factsStore memory.FactsStore
	if a.Facts != nil {
		factsStore = a.Facts
	}
	if ad, ok := a.ChatMemory.(*memory.Adapter); ok && ad != nil {
		ad.SetCompressor(compressor)
		ad.SetSessions(sessions)
		ad.SetFacts(factsStore)
		if a.Episodic != nil {
			ad.SetEpisodic(a.Episodic)
		}
		a.setMemory(ad)
		a.wireRecallRanker()
	a.wireProceduralMemory()
	a.wireConsolidator()
	a.wireRetrievalGate()
	return
}
ad := memory.NewAdapter(memory.AdapterConfig{
		Compressor: compressor,
		Sessions:   sessions,
		Evidence:   a.Evidence,
		Facts:      factsStore,
		Episodic:   a.Episodic,
	})
	a.ChatMemory = ad
	a.setMemory(ad)
	a.wireRecallRanker()
	a.wireProceduralMemory()
	a.wireConsolidator()
	a.wireRetrievalGate()
}

func (a *App) wireRetrievalGate() {
	if a == nil || a.Config == nil || a.Agent == nil {
		return
	}
	aux := a.Config.EffectiveAuxiliaryCompression()
	provider, err := llm.BuildProviderFromLLMFields(aux.Provider, aux.TokenKey, aux.Model, nil, "", aux.BaseURL, nil)
	if err != nil {
		return
	}
	var policy llm.Policy
	if a.Gateway != nil {
		policy = a.Gateway.Policy()
	}
	a.Agent.SetRetrievalGate(provider, policy, 4)
}

func (a *App) wireProceduralMemory() {
	if a == nil || a.Agent == nil {
		return
	}
	root := findProjectRoot()
	dirs := []string{filepath.Join(root, "skills")}
	if a.Workspace != "" {
		dirs = append(dirs, filepath.Join(a.Workspace, "skills"))
	}
	a.SkillLoader = procedural.NewLoader(dirs...)
	if a.Config != nil {
		skillsCfg := a.Config.EffectiveSkills()
		a.SkillLoader.SetPolicy(procedural.PolicyFromConfig(&skillsCfg))
	}
	a.Agent.SetSkillLoader(a.SkillLoader, 2)
}

func (a *App) wireConsolidator() {
	if a == nil || a.Config == nil {
		return
	}
	aux := a.Config.EffectiveAuxiliaryCompression()
	provider, err := llm.BuildProviderFromLLMFields(aux.Provider, aux.TokenKey, aux.Model, nil, "", aux.BaseURL, nil)
	if err != nil {
		return
	}
	var policy llm.Policy
	if a.Gateway != nil {
		policy = a.Gateway.Policy()
	}
	a.Consolidator = &consolidation.Distiller{
		Provider: provider,
		Policy:   policy,
		Facts:    a.Facts,
		Episodic: a.Episodic,
		EveryN:   3,
	}
}

func (a *App) setMemory(m memport.Port) {
	if a.Agent != nil {
		a.Agent.SetMemory(m)
	}
}

func (a *App) wireCognition() {
	if a == nil || a.Agent == nil || a.Config == nil {
		return
	}
	adv := a.Config.EffectiveAdvisor()
	if !adv.Enabled || adv.BaseURL == "" {
		a.Agent.SetCognition(cognition.Defaults())
		return
	}
	client := cognition.NewAdvisorClient(cognition.AdvisorConfig{
		BaseURL: adv.BaseURL, Timeout: adv.Timeout,
	})
	a.Agent.SetCognition(cognition.BundleWithAdvisor(client, adv.Ranker, adv.Evaluator))
}

func (a *App) wireRecallRanker() {
	if a == nil || a.Agent == nil {
		return
	}
	ad, ok := a.ChatMemory.(*memory.Adapter)
	if !ok || ad == nil {
		return
	}
	agentRef := a.Agent
	ad.SetSessionRanker(func(ctx context.Context, hits []memport.RecallHit) ([]memport.RecallHit, error) {
		return agentRef.RankRecallHits(ctx, hits)
	})
}

func (a *App) buildModelPolicy(thinkingOn bool, maxTokens int, temp float64) llm.Policy {
	base := llm.NewConfigPolicy(llm.ConfigPolicyInput{
		Temperature: temp, MaxTokens: maxTokens,
		CompressTemperature: 0.2,
		CompressMaxTokens:   maxTokens,
	})
	complexMin := maxTokens
	if thinkingOn && complexMin < 8192 {
		complexMin = 8192
	}
	return llm.ComplexityPolicy{
		Inner:               base,
		ComplexMinTokens:    complexMin,
		ToolSchemaThreshold: 0, // TaskComplex only; avoids 82-tool chat inflating max_tokens
	}
}

func (a *App) wireCompressor() { a.wireChatMemory() }

// Skills is the registry of runnable skills (built-in + any registered at runtime).
var DefaultSkills = skills.Default()

// RunPreMarket executes the premarket_market skill workflow.
// Kept for backward compatibility; new callers should use RunSkill.
func (a *App) RunPreMarket(skill string) (workflow.RunResult, error) {
	return a.RunSkill(skill)
}

// RunSkill executes a named skill workflow looked up in the skill registry.
// Returns an error if the skill is not registered.
func (a *App) RunSkill(skill string) (workflow.RunResult, error) {
	return a.RunSkillContext(context.Background(), skill)
}

// SkillRunOptions carries per-run inputs for signal-triggered skills.
type SkillRunOptions struct {
	Intraday     *workflow.IntradayInput
	MCPToken     string
	Market       string
	ReportDate   string
	NotifyFeishu bool
}

// RunSkillContext executes a named skill with cancellation propagated to tools
// and optional LLM synthesis.
func (a *App) RunSkillContext(ctx context.Context, skill string, runOpts ...SkillRunOptions) (workflow.RunResult, error) {
	var opts SkillRunOptions
	if len(runOpts) > 0 {
		opts = runOpts[0]
	}
	if skill == "premarket_stock" {
		market := workflow.NormalizeMarket(opts.Market)
		if market == "" {
			return workflow.RunResult{}, fmt.Errorf("premarket_stock requires market=CN|HK|US")
		}
		return a.runPreMarketStockForMarket(ctx, market, opts)
	}
	if skill == "postmarket_stock" {
		market := workflow.NormalizeMarket(opts.Market)
		if market == "" {
			return workflow.RunResult{}, fmt.Errorf("postmarket_stock requires market=CN|HK|US")
		}
		return a.runPostMarketStockForMarket(ctx, market, opts)
	}
	phaseA, perStock, err := a.resolveSkillSteps(skill, opts.Market)
	if err != nil {
		return workflow.RunResult{}, err
	}
	return a.runSkillWithSteps(ctx, skill, phaseA, perStock, opts)
}

func (a *App) resolveSkillSteps(skill, market string) ([]workflow.Step, []workflow.Step, error) {
	switch skill {
	case "premarket_market":
		m := workflow.NormalizeMarket(market)
		if m == "" {
			return nil, nil, fmt.Errorf("premarket_market requires market=CN|HK|US")
		}
		return workflow.MarketPhaseSteps(m), nil, nil
	case "premarket_stock":
		m := workflow.NormalizeMarket(market)
		if m == "" {
			return nil, nil, fmt.Errorf("premarket_stock requires market=CN|HK|US")
		}
		return workflow.StockPhaseASteps(m), workflow.PerStockSteps(), nil
	default:
		spec, ok := DefaultSkills.Get(skill)
		if !ok {
			return nil, nil, fmt.Errorf("unknown skill: %s (run 'geegoo skills list')", skill)
		}
		if spec.PhaseA == nil {
			return nil, nil, fmt.Errorf("skill %s has no step functions defined", skill)
		}
		phaseA := spec.PhaseA()
		var perStock []workflow.Step
		if spec.PerStock != nil {
			perStock = spec.PerStock()
		}
		if len(phaseA) == 0 && len(perStock) == 0 {
			return nil, nil, fmt.Errorf("skill %s is registered but has no executable steps", skill)
		}
		return phaseA, perStock, nil
	}
}

func (a *App) runSkillWithSteps(ctx context.Context, skill string, phaseA, perStock []workflow.Step, opts SkillRunOptions) (workflow.RunResult, error) {
	sessionID := newSessionID()
	a.EventBus.Emit("RunStarted", map[string]any{"session_id": sessionID, "skill": skill, "market": opts.Market})
	working, err := a.Working.Create(sessionID, skill)
	if err != nil {
		return workflow.RunResult{}, err
	}
	workflow.SeedMarketWorking(working, opts.Market)
	workflow.SeedReportDate(working, opts.ReportDate)
	if skill == "intraday_stock" {
		in := workflow.IntradayInputFromEnv()
		if opts.Intraday != nil {
			in = *opts.Intraday
		}
		workflow.SeedIntradayWorking(working, in)
		perStock = workflow.IntradayPerStockStepsForWorking(working)
	}
	if err := a.Working.Save(working); err != nil {
		return workflow.RunResult{}, err
	}
	toolCtx := a.ToolContextWithContext(ctx, sessionID)
	if strings.TrimSpace(opts.MCPToken) != "" {
		toolCtx.MCPToken = strings.TrimSpace(opts.MCPToken)
	}
	result := a.Workflow.Run(sessionID, skill, phaseA, perStock, toolCtx, working)
	a.emitSkillRunResult(sessionID, skill, result)
	return result, nil
}

func (a *App) runPreMarketStockForMarket(ctx context.Context, market string, opts SkillRunOptions) (workflow.RunResult, error) {
	return a.runStockForMarketUsers(ctx, "premarket_stock", market, opts, workflow.StockPhaseASteps(market), workflow.PerStockSteps())
}

func (a *App) runPostMarketStockForMarket(ctx context.Context, market string, opts SkillRunOptions) (workflow.RunResult, error) {
	return a.runStockForMarketUsers(ctx, "postmarket_stock", market, opts, workflow.PostMarketPhaseASteps(), workflow.PostMarketPerStockSteps())
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
	if !ok || spec.PhaseA == nil {
		return workflow.RunResult{}, fmt.Errorf("unsupported checkpoint skill: %s", cp.Skill)
	}
	working, err := a.Working.Load(sessionID)
	if err != nil {
		return workflow.RunResult{}, err
	}
	if working == nil {
		return workflow.RunResult{}, fmt.Errorf("working state not found for session: %s", sessionID)
	}
	phaseA, perStock, err := a.resolveSkillSteps(cp.Skill, working.Market)
	if err != nil {
		return workflow.RunResult{}, err
	}
	if len(phaseA) == 0 && len(perStock) == 0 {
		return workflow.RunResult{}, fmt.Errorf("checkpoint skill %s has no executable steps", cp.Skill)
	}
	if cp.Status == "completed" || working.Phase == "done" {
		return workflow.RunResult{SessionID: sessionID, Status: "completed", Working: working}, nil
	}
	toolCtx := a.ToolContextWithContext(ctx, sessionID)
	result := a.Workflow.RunFrom(sessionID, cp.Skill, phaseA, perStock, toolCtx, working, cp.Step)
	a.emitSkillRunResult(sessionID, cp.Skill, result)
	return result, nil
}

func (a *App) emitSkillRunResult(sessionID, skill string, result workflow.RunResult) {
	if a == nil || a.EventBus == nil {
		return
	}
	payload := map[string]any{
		"session_id": sessionID,
		"skill":      skill,
		"status":     result.Status,
	}
	if result.LastError != "" {
		payload["error"] = result.LastError
	}
	if result.Supervisor != nil {
		payload["verdict"] = string(result.Supervisor.Verdict)
	}
	if result.OK() {
		a.EventBus.Emit("RunCompleted", payload)
		return
	}
	a.EventBus.Emit("RunFailed", payload)
}

// ToolContext builds execution context for the current session.
func (a *App) ToolContext(sessionID string) tools.Context {
	return a.ToolContextWithContext(context.Background(), sessionID)
}

// ToolContextWithContext builds execution context for the current session.
func (a *App) ToolContextWithContext(ctx context.Context, sessionID string) tools.Context {
	tc := tools.Context{
		Ctx: ctx, SessionID: sessionID, MCPToken: a.Config.MCPToken(), DryRun: a.Config.DryRun,
		WorkspaceRoot: a.Workspace, StateStore: a.State,
		Hooks: a.Hooks,
	}
	if a.EventBus != nil {
		tc.EventBus = a.EventBus
	}
	return tc
}

func buildHooks(h config.HooksConfig) *tools.HookRunner {
	if len(h.ToolBefore) == 0 && len(h.ToolAfter) == 0 {
		return nil
	}
	timeout := time.Duration(h.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &tools.HookRunner{
		ToolBefore: append([]string(nil), h.ToolBefore...),
		ToolAfter:  append([]string(nil), h.ToolAfter...),
		FailClosed: h.FailClosed,
		Timeout:    timeout,
	}
}

// ChatToolNames returns registry tools enabled for interactive chat
// (filtered by config chat_toolsets).
func (a *App) ChatToolNames() []string {
	if a == nil {
		return nil
	}
	var ids []string
	if a.Config != nil {
		ids = a.Config.EffectiveChatToolsets()
	}
	return tools.RegisteredChatToolNamesFor(a.Registry, ids)
}

// EndpointSummary prints GeeGoo service endpoints.
func (a *App) EndpointSummary() string {
	return fmt.Sprintf(
		"GeeGooBot mcp-api %s | GeeGooSignal catalog %s | GeeGooData %s",
		a.Config.EffectiveMCPURL(), a.Config.SignalCatalogURL(), a.Config.DataHTTPURL(),
	)
}

// ProjectRoot returns the directory used to locate bundled skills/ and config assets.
func (a *App) ProjectRoot() string {
	return findProjectRoot()
}

// ProjectRoot returns the directory used to locate bundled skills/ and config assets.
func ProjectRoot() string {
	return findProjectRoot()
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "skills", "premarket_market", "SKILL.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
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
