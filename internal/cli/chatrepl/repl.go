package chatrepl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	prompt "github.com/c-bata/go-prompt"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Repl is interactive geegoo chat (Python chat_repl.py parity).
type Repl struct {
	App          *app.App
	ConfigPath   string
	Chat         *chatsession.ChatSession
	SessionStore *chatsession.ChatSessionStore
	Session      *runtime.Session
	Registry     *tools.Registry
	Loop         *runtime.ReActLoop
	UI           *chatui.ChatUI
	DryRun       bool
	Verbose      bool
	StepLog      []runtime.StepRecord
	InstallDir   string
	ProjectRoot  string
	stdin        io.Reader
	stdout       io.Writer
}

// New builds a chat REPL.
func New(application *app.App, configPath string, dryRun bool, stdout io.Writer) (*Repl, error) {
	return NewWithSession(application, configPath, "", dryRun, stdout)
}

// NewWithSession builds a REPL, optionally resuming an existing chat session.
func NewWithSession(application *app.App, configPath string, sessionID string, dryRun bool, stdout io.Writer) (*Repl, error) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if application.State == nil {
		return nil, fmt.Errorf("state store not configured")
	}
	store := chatsession.NewChatSessionStore(application.State)
	var chat *chatsession.ChatSession
	var err error
	if sessionID != "" {
		chat, err = store.Load(sessionID)
		if err != nil {
			return nil, err
		}
		if chat == nil {
			return nil, fmt.Errorf("chat session not found: %s", sessionID)
		}
	} else {
		chat, err = store.Create()
		if err != nil {
			return nil, err
		}
	}
	ui := chatui.New(stdout)
	r := &Repl{
		App: application, ConfigPath: configPath, Chat: chat, SessionStore: store,
		Session: &runtime.Session{
			ID: chat.ID, Messages: chat.RuntimeMessages(),
			StepCounter: chat.StepCounter, CreatedAt: chat.CreatedAt,
		},
		Registry: application.Registry, Loop: application.Loop, UI: ui,
		DryRun: dryRun || application.Config.DryRun, Verbose: true,
		stdout: stdout, stdin: os.Stdin, InstallDir: findInstallDir(), ProjectRoot: findProjectRoot(),
	}
	for _, rec := range chat.StepRecords {
		r.StepLog = append(r.StepLog, runtime.StepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	r.attachProgress()
	return r, nil
}

func findInstallDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home + string(os.PathSeparator) + ".geegoo" + string(os.PathSeparator) + "geegoo-agent"
	}
	return ""
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func (r *Repl) attachProgress() {
	r.Loop.SetProgress(func(event string, data map[string]any) {
		if r.Verbose {
			r.UI.EmitProgress(event, data)
		}
	})
}

// RunSingle handles one --message turn with full UI.
func (r *Repl) RunSingle(message string) int {
	r.printBanner()
	r.UI.PrintUser(message)
	result := r.runTurn(message)
	r.UI.PrintAssistant(result.AssistantText)
	r.printFooter(result)
	if result.Failed {
		return 1
	}
	return 0
}

// Run interactive loop.
func (r *Repl) Run() int {
	r.printBanner()
	for {
		line, err := r.readLine()
		if err != nil {
			r.UI.PrintInfo("\n再见。")
			return 0
		}
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "/") {
			if r.handleSlash(text) {
				return 0
			}
			continue
		}
		r.UI.PrintUser(text)
		result := r.runTurn(text)
		r.UI.PrintAssistant(result.AssistantText)
		r.printFooter(result)
	}
}

func (r *Repl) printBanner() {
	cfg := r.App.Config
	provider := llm.ProviderName(cfg.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	preset := llm.Presets[provider]
	model := llm.ResolveModel(provider, cfg.LLM.Model)
	workspace, _ := cfg.ResolveOutputDir()
	r.UI.PrintBanner(chatui.BannerOptions{
		SessionID: r.Chat.ID, Provider: preset.Label, Model: model,
		Registry: r.Registry, ToolNames: tools.RegisteredChatToolNames(r.Registry),
		Thinking: llm.ResolveThinkingEnabled(provider, model, cfg.LLM.Thinking),
		DryRun: r.DryRun, Workspace: workspace, InstallDir: r.InstallDir,
		ProjectRoot: r.ProjectRoot,
		APIHosts: chatui.APIHostsFromConfig(
			cfg.EffectiveMCPURL(), cfg.SignalCatalogURL(), cfg.DataHTTPURL(),
		),
		Revision: chatui.ResolveRevision(r.InstallDir),
	})
}

func (r *Repl) printFooter(result runtime.TurnResult) {
	cfg := r.App.Config
	provider := llm.ProviderName(cfg.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	model := llm.ResolveModel(provider, cfg.LLM.Model)
	thinking := llm.ResolveThinkingEnabled(provider, model, cfg.LLM.Thinking)
	r.UI.PrintTurnFooter(model, thinking, r.DryRun, len(r.StepLog))
}

func (r *Repl) runTurn(text string) runtime.TurnResult {
	r.Chat.SyncChatSystemPrompt()
	r.Session.Messages = r.Chat.RuntimeMessages()
	schemas := r.Registry.Schemas(tools.RegisteredChatToolNames(r.Registry))
	ctx := r.App.ToolContext(r.Session.ID)
	ctx.DryRun = r.DryRun
	result := r.Loop.RunTurn(r.Session, text, ctx, schemas)
	r.StepLog = append(r.StepLog, result.StepRecords...)
	newRecords := make([]chatsession.ChatStepRecord, 0, len(result.StepRecords))
	for _, rec := range result.StepRecords {
		newRecords = append(newRecords, chatsession.ChatStepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	r.Chat.SyncFromRuntime(r.Session.Messages, r.Session.StepCounter, newRecords)
	_ = r.SessionStore.Save(r.Chat)
	return result
}

func (r *Repl) saveSessionClosed() {
	r.Chat.Status = "closed"
	_ = r.SessionStore.Save(r.Chat)
}

func (r *Repl) readLine() (string, error) {
	if r.UI.IsPlain() {
		r.UI.PrintPrompt()
		reader := bufio.NewReader(r.stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	r.UI.PrintPrompt()
	completer := prompt.Completer(func(d prompt.Document) []prompt.Suggest {
		text := d.TextBeforeCursor()
		if !strings.HasPrefix(strings.TrimLeft(text, " "), "/") {
			return nil
		}
		prefix := strings.TrimLeft(text, " ")
		var out []prompt.Suggest
		for _, item := range chatui.SlashCommands {
			if strings.HasPrefix(item.Command, prefix) {
				out = append(out, prompt.Suggest{Text: item.Command, Description: item.Description})
			}
		}
		return out
	})
	line := prompt.Input("", completer,
		prompt.OptionTitle("geegoo chat"),
		prompt.OptionPrefix(""),
		prompt.OptionShowCompletionAtStart(),
	)
	return line, nil
}

func (r *Repl) handleSlash(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/exit", "/quit":
		r.saveSessionClosed()
		r.UI.PrintInfo("会话已保存。")
		return true
	case "/help":
		r.UI.PrintHelp(chatui.BuildHelpText())
	case "/session":
		cfg := r.App.Config
		provider := llm.ProviderName(cfg.LLM.Provider)
		if provider == "" {
			provider = llm.ProviderDeepSeek
		}
		model := llm.ResolveModel(provider, cfg.LLM.Model)
		preset := llm.Presets[provider]
		think := "off"
		if llm.ResolveThinkingEnabled(provider, model, cfg.LLM.Thinking) {
			think = "on"
		}
		r.UI.PrintInfo(fmt.Sprintf(
			"session=%s messages=%d steps=%d dry_run=%v llm=%s/%s verbose=%v think=%s",
			r.Chat.ID, len(r.Chat.Messages), len(r.Chat.StepRecords), r.DryRun,
			preset.Label, model, r.Verbose, think,
		))
	case "/tools":
		r.printTools()
	case "/trace":
		limit := 10
		if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil {
				limit = n
			}
		}
		r.printTrace(limit)
	case "/flow":
		limit := 15
		if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil {
				limit = n
			}
		}
		r.printFlow(limit)
	case "/dry-run":
		if len(args) == 0 || (args[0] != "on" && args[0] != "off") {
			fmt.Fprintf(r.stdout, "用法: /dry-run on|off\n")
			break
		}
		r.DryRun = args[0] == "on"
		fmt.Fprintf(r.stdout, "dry_run=%v\n", r.DryRun)
	case "/verbose":
		if len(args) == 0 || (args[0] != "on" && args[0] != "off") {
			fmt.Fprintf(r.stdout, "用法: /verbose on|off\n")
			break
		}
		r.Verbose = args[0] == "on"
		r.attachProgress()
		fmt.Fprintf(r.stdout, "verbose=%v\n", r.Verbose)
	case "/model":
		if len(args) > 0 {
			r.setModel(args[0])
		} else {
			r.printModels()
		}
	case "/think":
		r.handleThink(args)
	case "/run":
		skill := "pre_market"
		if len(args) > 0 {
			skill = args[0]
		}
		r.runWorkflow(skill)
	default:
		fmt.Fprintf(r.stdout, "未知命令: %s，输入 /help\n", cmd)
	}
	return false
}

func (r *Repl) printTools() {
	names := tools.RegisteredChatToolNames(r.Registry)
	descriptions := map[string]string{}
	for _, name := range names {
		if t, ok := r.Registry.Get(name); ok {
			descriptions[name] = t.Description
		}
	}
	fmt.Fprintf(r.stdout, "%s\n", tools.FormatToolsListing(names, descriptions))
}

func (r *Repl) printTrace(limit int) {
	if len(r.StepLog) == 0 {
		fmt.Fprintln(r.stdout, "（暂无步骤记录）")
		return
	}
	records := r.StepLog
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}
	for _, rec := range records {
		if rec.Kind == "tool" {
			fmt.Fprintf(r.stdout, "  #%d [%s] %s %s: %s\n", rec.Step, rec.Kind, rec.ToolName, rec.ToolStatus, truncateText(rec.Summary, 100))
		} else {
			fmt.Fprintf(r.stdout, "  #%d [%s] %s\n", rec.Step, rec.Kind, truncateText(rec.Summary, 100))
		}
	}
}

func (r *Repl) printFlow(limit int) {
	history := r.App.EventBus.History
	if len(history) == 0 {
		fmt.Fprintln(r.stdout, "（暂无事件）")
		return
	}
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}
	for _, rec := range history {
		parts := make([]string, 0, len(rec.Payload))
		for k, v := range rec.Payload {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fmt.Fprintf(r.stdout, "  %s: %s\n", rec.Event, strings.Join(parts, ", "))
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func (r *Repl) printModels() {
	provider := llm.ProviderName(r.App.Config.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	preset := llm.Presets[provider]
	active := llm.ResolveModel(provider, r.App.Config.LLM.Model)
	fmt.Fprintf(r.stdout, "当前: %s / %s\n", preset.Label, active)
	models := llm.ListProviderModels(provider)
	if len(models) == 0 {
		fmt.Fprintf(r.stdout, "（该提供商无预设列表，可用 /model <model_id> 手动指定）\n")
		return
	}
	for i, m := range models {
		mark := ""
		if m.ID == active {
			mark = " *"
		}
		fmt.Fprintf(r.stdout, "  %d. %s — %s%s\n", i+1, m.ID, m.Description, mark)
	}
	fmt.Fprintf(r.stdout, "切换: /model <序号> 或 /model <model_id>\n")
}

func (r *Repl) setModel(choice string) {
	provider := llm.ProviderName(r.App.Config.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	resolved, err := llm.PickModel(provider, choice, r.App.Config.LLM.Model)
	if err != nil {
		fmt.Fprintf(r.stdout, "切换失败: %v\n", err)
		return
	}
	r.App.Config.LLM.Model = resolved
	r.persistConfigField(func(llmCfg *config.LLMConfig) {
		llmCfg.Model = resolved
	})
	if err := r.App.RebuildGateway(); err != nil {
		fmt.Fprintf(r.stdout, "警告: 重建 LLM 失败: %v\n", err)
	}
	fmt.Fprintf(r.stdout, "已切换模型: %s / %s\n", llm.Presets[provider].Label, resolved)
}

func (r *Repl) handleThink(args []string) {
	provider := llm.ProviderName(r.App.Config.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	model := llm.ResolveModel(provider, r.App.Config.LLM.Model)
	if !llm.ModelSupportsThinking(provider, model) {
		r.UI.PrintError("当前模型不支持思考模式，请 /model deepseek-v4-pro 或 v4-flash")
		return
	}
	if len(args) == 0 || (args[0] != "on" && args[0] != "off" && args[0] != "auto") {
		active := "off"
		if llm.ResolveThinkingEnabled(provider, model, r.App.Config.LLM.Thinking) {
			active = "on"
		}
		if r.App.Config.LLM.Thinking == nil {
			active = "auto"
		}
		r.UI.PrintInfo(fmt.Sprintf("思考模式: %s（用法: /think on|off|auto）", active))
		return
	}
	var enabled *bool
	switch args[0] {
	case "on":
		v := true
		enabled = &v
	case "off":
		v := false
		enabled = &v
	case "auto":
		enabled = nil
	}
	r.App.Config.LLM.Thinking = enabled
	r.persistConfigField(func(llmCfg *config.LLMConfig) {
		llmCfg.Thinking = enabled
	})
	if err := r.App.RebuildGateway(); err != nil {
		r.UI.PrintError("重建 LLM 失败: " + err.Error())
		return
	}
	state := args[0]
	if enabled != nil {
		if *enabled {
			state = "on"
		} else {
			state = "off"
		}
	} else {
		state = "auto"
	}
	r.UI.PrintInfo("思考模式已设为: " + state)
}

func (r *Repl) persistConfigField(mutate func(*config.LLMConfig)) {
	if r.ConfigPath == "" {
		return
	}
	raw, err := os.ReadFile(r.ConfigPath)
	if err != nil {
		return
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	llmRaw, _ := doc["llm"].(map[string]any)
	if llmRaw == nil {
		llmRaw = map[string]any{}
		doc["llm"] = llmRaw
	}
	cfg := r.App.Config.LLM
	mutate(&cfg)
	if cfg.Model != "" {
		llmRaw["model"] = cfg.Model
	}
	if cfg.Thinking == nil {
		delete(llmRaw, "thinking")
	} else {
		llmRaw["thinking"] = *cfg.Thinking
	}
	out, _ := json.MarshalIndent(doc, "", "  ")
	_ = os.WriteFile(r.ConfigPath, append(out, '\n'), 0o600)
}

func (r *Repl) runWorkflow(skill string) {
	fmt.Fprintf(r.stdout, "启动 workflow: %s ...\n", skill)
	prev := r.App.Config.DryRun
	r.App.Config.DryRun = r.DryRun
	result, err := r.App.RunPreMarket(skill)
	r.App.Config.DryRun = prev
	if err != nil {
		fmt.Fprintf(r.stdout, "workflow 失败: %v\n", err)
		return
	}
	fmt.Fprintf(r.stdout, "workflow 完成: session=%s status=%s\n", result.SessionID, result.Status)
	if result.LastError != "" {
		fmt.Fprintf(r.stdout, "  error: %s\n", result.LastError)
	}
	workspace, _ := r.App.Config.ResolveOutputDir()
	fmt.Fprintf(r.stdout, "  查看 execution-log: %s\n", workspace)
	fmt.Fprintf(r.stdout, "  使用 /flow 查看本次 workflow 触发的 Tool 事件\n")
}
