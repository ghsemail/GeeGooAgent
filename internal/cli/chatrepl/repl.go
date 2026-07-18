package chatrepl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	prompt "github.com/c-bata/go-prompt"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/cli/flowview"
	"github.com/ghsemail/GeeGooAgent/internal/cli/progress"
	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
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
	SessionStore chatsession.SessionStore
	Session      *runtime.Session
	Registry *tools.Registry
	UI       *chatui.ChatUI
	Progress     progress.Sink // optional override; default UI
	DryRun       bool
	Verbose      bool
	ChatToolsets []string // session override; nil → config defaults
	StepLog      []runtime.StepRecord
	InstallDir   string
	ProjectRoot  string
	curCancel    context.CancelFunc
	stdin        io.Reader
	stdout       io.Writer
	tty          *ttyState
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
	if application.State == nil && application.DB == nil {
		return nil, fmt.Errorf("no state store or database configured")
	}
	var store chatsession.SessionStore
	if application.DB != nil {
		store = chatsession.NewSQLiteSessionStore(application.DB)
	} else {
		store = chatsession.NewChatSessionStore(application.State)
	}
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
	parentID, lineageRoot, generation := chat.LineageFromMetadata()
	r := &Repl{
		App: application, ConfigPath: configPath, Chat: chat, SessionStore: store,
		Session: &runtime.Session{
			ID: chat.ID, Messages: chat.RuntimeMessages(),
			StepCounter: chat.StepCounter, CreatedAt: chat.CreatedAt,
			ParentID: parentID, LineageRoot: lineageRoot, CompactionGeneration: generation,
		},
		Registry: application.Registry, UI: ui,
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
	r.attachApproval()
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
	sink := r.Progress
	if sink == nil {
		sink = r.UI
	}
	live := chatsession.NewLivePublisher(r.App.State, r.Chat.ID)
	r.App.Agent.SetProgress(func(event string, data map[string]any) {
		if live != nil {
			live.Emit(event, data)
		}
		switch event {
		case "stream_delta", "turn_start", "reply_start", "thinking_start", "thinking_stop":
			sink.EmitProgress(event, data)
			return
		case "llm_tools", "tool_start", "error":
			// Abort typewriter even when verbose is off; optionally show tool lines.
			if r.Verbose {
				sink.EmitProgress(event, data)
			} else if ui, ok := sink.(*chatui.ChatUI); ok {
				ui.AbortStreamReply()
				if event == "error" {
					if msg, _ := data["message"].(string); msg != "" {
						ui.PrintError(msg)
					}
				}
			} else if event == "error" {
				sink.EmitProgress(event, data)
			}
			return
		}
		if r.Verbose {
			sink.EmitProgress(event, data)
		}
	})
}

// SetApprovalFn replaces the write-tool approval prompt (used by TUI modal).
func (r *Repl) SetApprovalFn(fn runtime.ApprovalFunc) {
	r.App.Agent.SetApproval(fn)
}

// SetProgressSink replaces the live progress target and re-wires the agent.
func (r *Repl) SetProgressSink(sink progress.Sink) {
	r.Progress = sink
	r.attachProgress()
}

// RunTurn executes one user message (exported for TUI host).
func (r *Repl) RunTurn(text string) runtime.TurnResult {
	return r.runTurn(text)
}

// CancelTurn cancels the in-flight turn context, if any.
func (r *Repl) CancelTurn() {
	if r.curCancel != nil {
		r.curCancel()
	}
}

// CloseSession marks the chat closed and persists.
func (r *Repl) CloseSession() {
	r.saveSessionClosed()
}

func (r *Repl) attachApproval() {
	r.App.Agent.SetApproval(r.promptToolApproval)
}

func (r *Repl) promptToolApproval(toolName string, args map[string]any) bool {
	restoreTTY(r.tty)
	argsJSON, _ := json.Marshal(args)
	summary := string(argsJSON)
	if len(summary) > 240 {
		summary = summary[:237] + "..."
	}
	r.UI.PrintInfo(fmt.Sprintf(
		"写操作确认: %s\n参数: %s\n输入 y/yes 执行，其他键跳过",
		toolName, summary,
	))
	reader := bufio.NewReader(r.stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// RunSingle handles one --message turn with full UI.
func (r *Repl) RunSingle(message string) int {
	r.printBanner()
	r.UI.PrintUser(message)
	result := r.runTurn(message)
	r.finishAssistant(result)
	r.printFooter(result)
	if result.Failed {
		return 1
	}
	return 0
}

// Run interactive loop.
func (r *Repl) Run() int {
	r.tty = saveTTY()
	defer restoreTTY(r.tty)
	defer func() {
		if r.App != nil {
			_ = r.App.Close()
		}
	}()

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
		r.finishAssistant(result)
		r.printFooter(result)
	}
}

func (r *Repl) finishAssistant(result runtime.TurnResult) {
	if r.UI.FinishAssistantStream() {
		return
	}
	if strings.TrimSpace(result.AssistantText) == "" {
		return
	}
	r.UI.PrintAssistant(result.AssistantText)
}

// turnCtx returns a per-turn context that is cancelled on SIGINT (Ctrl+C).
// Cancelling mid-turn aborts in-flight LLM and tool calls; the next turn
// gets a fresh context so the user can continue chatting afterwards.
func (r *Repl) turnCtx() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	r.curCancel = stop
	return ctx
}

func (r *Repl) printBanner() {
	r.UI.PrintBanner(r.BannerOptions())
}

// BannerOptions builds Hermes-style welcome panel metadata.
func (r *Repl) BannerOptions() chatui.BannerOptions {
	cfg := r.App.Config
	provider := llm.ProviderName(cfg.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	preset := llm.Presets[provider]
	model := r.App.EffectiveLLMModel()
	workspace, _ := cfg.ResolveOutputDir()
	return chatui.BannerOptions{
		SessionID: r.Chat.ID, Provider: preset.Label, Model: model,
		Registry: r.Registry, ToolNames: r.chatToolNames(),
		Thinking: llm.ResolveThinkingEnabled(provider, model, cfg.LLM.Thinking),
		DryRun:   r.DryRun, Workspace: workspace, InstallDir: r.InstallDir,
		ProjectRoot: r.ProjectRoot,
		APIHosts: chatui.APIHostsFromConfig(
			cfg.EffectiveMCPURL(), cfg.SignalCatalogURL(), cfg.DataHTTPURL(),
		),
		Revision: chatui.ResolveRevision(r.InstallDir),
	}
}

func (r *Repl) printFooter(result runtime.TurnResult) {
	cfg := r.App.Config
	provider := llm.ProviderName(cfg.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	model := r.App.EffectiveLLMModel()
	thinking := llm.ResolveThinkingEnabled(provider, model, cfg.LLM.Thinking)
	r.UI.PrintTurnFooter(model, thinking, r.DryRun, len(r.StepLog))
}

func (r *Repl) runTurn(text string) runtime.TurnResult {
	r.Chat.SyncChatSystemPrompt()
	r.Session.Messages = r.Chat.RuntimeMessages()
	schemas := r.Registry.Schemas(r.chatToolNames())
	ctx := r.App.ToolContext(r.Session.ID)
	ctx.DryRun = r.DryRun
	// Chat is an interactive, user-facing entry point.  Mutating tools must
	// therefore pass through ApprovalGate instead of inheriting the workflow
	// default (non-interactive) policy from App.ToolContext.
	ctx.Interactive = true
	turnCtx := r.turnCtx()
	result := r.App.Agent.Run(turnCtx, r.Session, text, ctx, schemas)
	if r.curCancel != nil {
		r.curCancel()
		r.curCancel = nil
	}
	if result.Failed && (turnCtx.Err() != nil) {
		r.UI.PrintInfo("（本回合已中断，可继续输入下一句）")
	}
	r.StepLog = append(r.StepLog, result.StepRecords...)
	newRecords := make([]chatsession.ChatStepRecord, 0, len(result.StepRecords))
	for _, rec := range result.StepRecords {
		newRecords = append(newRecords, chatsession.ChatStepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	r.Chat.SyncFromRuntime(r.Session.Messages, r.Session.StepCounter, newRecords)
	r.Chat.SyncLineageFromRuntime(r.Session.ParentID, r.Session.LineageRoot, r.Session.CompactionGeneration)
	_ = r.SessionStore.Save(r.Chat)
	if pub := chatsession.NewLivePublisher(r.App.State, r.Chat.ID); pub != nil {
		pub.EndTurn()
	}
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
	line := prompt.Input("", completer, slashPromptOptions()...)
	// go-prompt TearDown often leaves the tty in a bad state; restore before next UI write.
	restoreTTY(r.tty)
	return line, nil
}

func slashPromptOptions() []prompt.Option {
	// Default go-prompt uses Cyan/Turquoise which renders as muddy brown-yellow
	// on many Windows/SSH terminals and fights our gold theme. Prefer dark panels.
	return []prompt.Option{
		prompt.OptionTitle("geegoo chat"),
		prompt.OptionPrefix(""),
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionInputTextColor(prompt.White),
		prompt.OptionPrefixTextColor(prompt.Yellow),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSuggestionTextColor(prompt.White),
		prompt.OptionDescriptionBGColor(prompt.Black),
		prompt.OptionDescriptionTextColor(prompt.LightGray),
		prompt.OptionSelectedSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionTextColor(prompt.Yellow),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSelectedDescriptionTextColor(prompt.LightGray),
		prompt.OptionPreviewSuggestionBGColor(prompt.DarkGray),
		prompt.OptionPreviewSuggestionTextColor(prompt.LightGray),
		prompt.OptionScrollbarBGColor(prompt.DarkGray),
		prompt.OptionScrollbarThumbColor(prompt.LightGray),
	}
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
		r.UI.PrintInfo("会话已保存。再见。")
		restoreTTY(r.tty)
		if r.App != nil {
			_ = r.App.Close()
		}
		// Force-exit: go-prompt can leave goroutines/tty in a state where a
		// normal return appears to hang the SSH session.
		os.Exit(0)
		return true
	case "/help":
		r.UI.PrintHelp(chatui.BuildHelpText())
	case "/session":
		cfg := r.App.Config
		provider := llm.ProviderName(cfg.LLM.Provider)
		if provider == "" {
			provider = llm.ProviderDeepSeek
		}
		model := r.App.EffectiveLLMModel()
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
	case "/toolsets":
		r.handleToolsets(args)
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

// HandleSlashCommand runs a slash command and captures stdout/UI output (TUI).
// Do not use for /exit — handleSlash may call os.Exit.
func (r *Repl) HandleSlashCommand(line string) (quit bool, output string) {
	if r == nil || r.UI == nil {
		return false, ""
	}
	var buf strings.Builder
	r.UI.WithPlainWriter(&buf, func() {
		oldStdout := r.stdout
		r.stdout = &buf
		quit = r.handleSlash(line)
		r.stdout = oldStdout
	})
	return quit, strings.TrimSpace(buf.String())
}

func (r *Repl) chatToolNames() []string {
	ids := r.ChatToolsets
	if ids == nil && r.App != nil && r.App.Config != nil {
		ids = r.App.Config.EffectiveChatToolsets()
	}
	return tools.RegisteredChatToolNamesFor(r.Registry, ids)
}

func (r *Repl) activeToolsetIDs() []string {
	if r.ChatToolsets != nil {
		normalized, err := tools.NormalizeToolsetIDs(r.ChatToolsets)
		if err == nil {
			return normalized
		}
	}
	if r.App != nil && r.App.Config != nil {
		normalized, _ := tools.NormalizeToolsetIDs(r.App.Config.EffectiveChatToolsets())
		return normalized
	}
	return tools.DefaultChatToolsetIDs()
}

func (r *Repl) printTools() {
	names := r.chatToolNames()
	descriptions := map[string]string{}
	for _, name := range names {
		if t, ok := r.Registry.Get(name); ok {
			descriptions[name] = t.Description
		}
	}
	fmt.Fprintf(r.stdout, "%s\n", tools.FormatToolsListing(names, descriptions))
}

func (r *Repl) handleToolsets(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.stdout, "%s\n", tools.FormatToolsetsListing(r.activeToolsetIDs()))
		return
	}
	raw := strings.Join(args, ",")
	raw = strings.ReplaceAll(raw, " ", "")
	if strings.EqualFold(raw, "default") || strings.EqualFold(raw, "all") {
		r.ChatToolsets = nil
		if r.App != nil && r.App.Config != nil {
			r.App.Config.ChatToolsets = nil
		}
		r.persistChatToolsets(nil)
		fmt.Fprintf(r.stdout, "已恢复默认 toolsets（%d tools）\n", len(r.chatToolNames()))
		return
	}
	parts := strings.Split(raw, ",")
	ids, err := tools.NormalizeToolsetIDs(parts)
	if err != nil {
		fmt.Fprintf(r.stdout, "切换失败: %v\n", err)
		return
	}
	r.ChatToolsets = ids
	if r.App != nil && r.App.Config != nil {
		r.App.Config.ChatToolsets = append([]string(nil), ids...)
	}
	r.persistChatToolsets(ids)
	fmt.Fprintf(r.stdout, "已切换 toolsets: %s（%d tools）\n", strings.Join(ids, ","), len(r.chatToolNames()))
}

func (r *Repl) persistChatToolsets(ids []string) {
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
	if len(ids) == 0 {
		delete(doc, "chat_toolsets")
	} else {
		arr := make([]any, len(ids))
		for i, id := range ids {
			arr[i] = id
		}
		doc["chat_toolsets"] = arr
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.ConfigPath, append(out, '\n'), 0o600)
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
		fmt.Fprintf(r.stdout, "  [%s] %s\n", rec.Event, flowview.Format(rec))
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func (r *Repl) printModels() {
	models, err := r.fetchCatalogModels()
	if err != nil {
		fmt.Fprintf(r.stdout, "无法拉取模型管理列表: %v\n", err)
		r.printProviderModelsFallback()
		return
	}
	if len(models) == 0 {
		fmt.Fprintf(r.stdout, "模型管理列表为空\n")
		return
	}
	activeID := llm.ActiveCatalogModelID(r.App.Config.LLM.CatalogModelID, models)
	activeName := ""
	for _, m := range models {
		if m.ModelID == activeID {
			activeName = llm.CatalogModelLabel(m)
			break
		}
	}
	if activeName == "" {
		activeName = activeID
	}
	if strings.TrimSpace(r.App.Config.LLM.CatalogModelID) == "" {
		fmt.Fprintf(r.stdout, "当前: %s（trading_operation 运营默认）\n", activeName)
	} else {
		fmt.Fprintf(r.stdout, "当前: %s\n", activeName)
	}
	for i, m := range models {
		mark := ""
		if m.ModelID == activeID {
			mark = " *"
		}
		fmt.Fprintf(r.stdout, "  %d. %s%s\n", i+1, llm.CatalogModelLabel(m), mark)
	}
	fmt.Fprintf(r.stdout, "切换: /model <序号> | /model <model_id> | /model default 恢复运营默认\n")
}

func (r *Repl) printProviderModelsFallback() {
	provider := llm.ProviderName(r.App.Config.LLM.Provider)
	if provider == "" {
		provider = llm.ProviderDeepSeek
	}
	preset := llm.Presets[provider]
	active := llm.ResolveModel(provider, r.App.Config.LLM.Model)
	fmt.Fprintf(r.stdout, "回退列表 — 当前: %s / %s\n", preset.Label, active)
	for i, m := range llm.ListProviderModels(provider) {
		mark := ""
		if m.ID == active {
			mark = " *"
		}
		fmt.Fprintf(r.stdout, "  %d. %s — %s%s\n", i+1, m.ID, m.Description, mark)
	}
}

func (r *Repl) fetchCatalogModels() ([]admin.ConfiguredModel, error) {
	if r.App == nil || r.App.Config == nil {
		return nil, fmt.Errorf("app not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	targets := make([]admin.QueryTarget, 0, len(r.App.Config.AdminModelQueryTargets()))
	for _, t := range r.App.Config.AdminModelQueryTargets() {
		targets = append(targets, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
	}
	docs, _, err := admin.ListModelsFromTargets(ctx, targets)
	return docs, err
}

func (r *Repl) setModel(choice string) {
	models, err := r.fetchCatalogModels()
	if err == nil && len(models) > 0 {
		r.setCatalogModel(choice, models)
		return
	}
	r.setProviderModel(choice)
}

func (r *Repl) setCatalogModel(choice string, models []admin.ConfiguredModel) {
	modelID, err := llm.PickCatalogModel(choice, models, r.App.Config.LLM.CatalogModelID)
	if err != nil {
		fmt.Fprintf(r.stdout, "切换失败: %v\n", err)
		return
	}
	r.App.Config.LLM.CatalogModelID = modelID
	useOps := true
	r.App.Config.LLM.UseOpsModel = &useOps
	if err := r.App.RebuildGateway(); err != nil {
		fmt.Fprintf(r.stdout, "警告: 重建 LLM 失败: %v\n", err)
	}
	r.persistLLMConfig()
	activeID := llm.ActiveCatalogModelID(modelID, models)
	label := activeID
	for _, m := range models {
		if m.ModelID == activeID {
			label = llm.CatalogModelLabel(m)
			break
		}
	}
	if modelID == "" {
		fmt.Fprintf(r.stdout, "已恢复运营默认模型: %s\n", label)
	} else {
		fmt.Fprintf(r.stdout, "已切换模型: %s\n", label)
	}
}

func (r *Repl) setProviderModel(choice string) {
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
	r.App.Config.LLM.CatalogModelID = ""
	useOps := false
	r.App.Config.LLM.UseOpsModel = &useOps
	r.persistLLMConfig()
	if err := r.App.RebuildGateway(); err != nil {
		fmt.Fprintf(r.stdout, "警告: 重建 LLM 失败: %v\n", err)
	}
	fmt.Fprintf(r.stdout, "已切换模型: %s / %s\n", llm.Presets[provider].Label, resolved)
}

func (r *Repl) persistLLMConfig() {
	r.persistConfigField(func(llmCfg *config.LLMConfig) {
		*llmCfg = r.App.Config.LLM
	})
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
	if cfg.CatalogModelID != "" {
		llmRaw["catalog_model_id"] = cfg.CatalogModelID
	} else {
		delete(llmRaw, "catalog_model_id")
	}
	if cfg.UseOpsModel != nil {
		llmRaw["use_ops_model"] = *cfg.UseOpsModel
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
	if result.Supervisor != nil {
		fmt.Fprintf(r.stdout, "  supervisor: %s\n", result.Supervisor.Summary())
	}
	fmt.Fprintf(r.stdout, "  使用 /flow 查看 Run/Tool/Synthesis 事件轨迹\n")
}
