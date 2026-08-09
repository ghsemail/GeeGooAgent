package gateway

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Config holds Runner options shared across platforms.
type Config struct {
	// AllowedUsers maps platform → set of user ids. Empty set + AllowAll → allow everyone.
	AllowedUsers map[Platform]map[string]struct{}
	AllowAll     map[Platform]bool
	HomeChannels map[Platform]HomeChannel
	DryRun       bool
}

// Runner owns adapters and routes inbound events into Agent.Run.
type Runner struct {
	App      *app.App
	Config   Config
	Adapters []PlatformAdapter
	Sessions *SessionMap

	mu     sync.Mutex
	chatMu map[string]*sync.Mutex // serialise per SessionKey
	seen   *dedupCache
}

// NewRunner constructs a gateway runner.
func NewRunner(application *app.App, sessions *SessionMap, cfg Config, adapters ...PlatformAdapter) *Runner {
	return &Runner{
		App:      application,
		Config:   cfg,
		Adapters: adapters,
		Sessions: sessions,
		chatMu:   map[string]*sync.Mutex{},
		seen:     newDedupCache(4096),
	}
}

// Start connects all configured adapters and blocks until ctx is done.
func (r *Runner) Start(ctx context.Context) error {
	if r == nil || r.App == nil {
		return fmtError("nil runner")
	}
	errCh := make(chan error, len(r.Adapters))
	var wg sync.WaitGroup
	started := 0
	for _, ad := range r.Adapters {
		if ad == nil || !ad.Configured() {
			continue
		}
		started++
		wg.Add(1)
		go func(a PlatformAdapter) {
			defer wg.Done()
			slog.Info("gateway: connecting", "platform", a.Platform())
			if err := a.Connect(ctx, r.handleInbound); err != nil && ctx.Err() == nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(ad)
	}
	if started == 0 {
		return fmtError("no configured IM adapters (set FEISHU_APP_ID / FEISHU_APP_SECRET)")
	}
	select {
	case <-ctx.Done():
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = r.Stop(stopCtx)
		wg.Wait()
		if ctx.Err() == context.Canceled {
			return nil
		}
		return ctx.Err()
	case err := <-errCh:
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = r.Stop(stopCtx)
		wg.Wait()
		return err
	}
}

// Stop disconnects all adapters.
func (r *Runner) Stop(ctx context.Context) error {
	var first error
	for _, ad := range r.Adapters {
		if ad == nil {
			continue
		}
		if err := ad.Disconnect(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Status returns adapter snapshots.
func (r *Runner) Status() []AdapterStatus {
	out := make([]AdapterStatus, 0, len(r.Adapters))
	for _, ad := range r.Adapters {
		if ad == nil {
			continue
		}
		out = append(out, ad.Status())
	}
	return out
}

func (r *Runner) handleInbound(ctx context.Context, ev InboundEvent) error {
	if ev.MessageID != "" && !r.seen.TryAdd(ev.MessageID) {
		slog.Debug("gateway: duplicate message dropped", "id", ev.MessageID)
		return nil
	}
	if !r.authorize(ev) {
		slog.Debug("gateway: user not allowed", "platform", ev.Platform, "user", ev.UserID, "chat", ev.ChatID)
		return nil
	}
	text := trimText(ev.Text)
	if text == "" {
		return nil
	}

	key := SessionKey(ev.Platform, ev.ChatID, ev.UserID)
	lock := r.lockFor(key)
	lock.Lock()
	defer lock.Unlock()

	adapter := r.adapter(ev.Platform)
	if adapter == nil {
		return fmtError("no adapter for " + string(ev.Platform))
	}

	reply, err := r.runAgentTurn(ctx, key, ev, text)
	if err != nil {
		slog.Error("gateway: agent turn failed", "err", err, "platform", ev.Platform)
		_ = adapter.SendText(ctx, OutboundText{ChatID: ev.ChatID, Text: "抱歉，处理消息时出错了。", ReplyToID: ev.MessageID})
		return err
	}
	if reply == "" {
		return nil
	}
	if r.Config.DryRun {
		slog.Info("gateway: dry-run reply", "chat", ev.ChatID, "text", truncate(reply, 200))
		return nil
	}
	return adapter.SendText(ctx, OutboundText{ChatID: ev.ChatID, Text: reply, ReplyToID: ev.MessageID})
}

func (r *Runner) authorize(ev InboundEvent) bool {
	if r.Config.AllowAll[ev.Platform] {
		return true
	}
	allowed := r.Config.AllowedUsers[ev.Platform]
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[ev.UserID]
	return ok
}

func (r *Runner) adapter(p Platform) PlatformAdapter {
	for _, ad := range r.Adapters {
		if ad != nil && ad.Platform() == p {
			return ad
		}
	}
	return nil
}

func (r *Runner) lockFor(key string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.chatMu[key]; ok {
		return m
	}
	m := &sync.Mutex{}
	r.chatMu[key] = m
	return m
}

func (r *Runner) runAgentTurn(ctx context.Context, key string, ev InboundEvent, text string) (string, error) {
	store, err := r.App.SessionStore()
	if err != nil {
		return "", err
	}
	chat, err := r.loadOrCreateChat(store, key, ev)
	if err != nil {
		return "", err
	}
	rt := agent.RuntimeSessionFromChat(chat)
	chat.SyncChatSystemPrompt()
	rt.Messages = chat.RuntimeMessages()

	toolNames := tools.RegisteredChatToolNamesFor(r.App.Registry, r.App.Config.EffectiveChatToolsets())
	schemas := r.App.Registry.Schemas(toolNames)
	toolCtx := r.App.ToolContext(rt.ID)
	toolCtx.DryRun = r.Config.DryRun || (r.App.Config != nil && r.App.Config.DryRun)
	toolCtx.Interactive = false // IM: no TUI clarify; mutating tools follow non-interactive policy

	result := r.App.Agent.Run(ctx, rt, text, toolCtx, schemas)
	newRecords := make([]chatsession.ChatStepRecord, 0, len(result.StepRecords))
	for _, rec := range result.StepRecords {
		newRecords = append(newRecords, chatsession.ChatStepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	agent.SyncChatFromRuntime(chat, rt, newRecords)
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["gateway_platform"] = string(ev.Platform)
	chat.Metadata["gateway_key"] = key
	chat.Metadata["gateway_chat_id"] = ev.ChatID
	chat.Metadata["gateway_user_id"] = ev.UserID
	if err := store.Save(chat); err != nil {
		slog.Warn("gateway: save session", "err", err)
	}
	if result.Failed && result.AssistantText == "" {
		return "", fmtError("agent turn failed")
	}
	return result.AssistantText, nil
}

func (r *Runner) loadOrCreateChat(store chatsession.SessionStore, key string, ev InboundEvent) (*chatsession.ChatSession, error) {
	if id, ok := r.Sessions.Get(key); ok && id != "" {
		chat, err := store.Load(id)
		if err != nil {
			return nil, err
		}
		if chat != nil {
			return chat, nil
		}
	}
	chat, err := store.Create()
	if err != nil {
		return nil, err
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["gateway_platform"] = string(ev.Platform)
	chat.Metadata["gateway_key"] = key
	chat.Metadata["gateway_chat_id"] = ev.ChatID
	chat.Metadata["gateway_user_id"] = ev.UserID
	chat.Title = "IM " + string(ev.Platform) + " " + ev.UserID
	if err := store.Save(chat); err != nil {
		return nil, err
	}
	if err := r.Sessions.Put(key, chat.ID); err != nil {
		slog.Warn("gateway: persist session map", "err", err)
	}
	return chat, nil
}

func trimText(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

type fmtError string

func (e fmtError) Error() string { return string(e) }

// dedupCache is an LRU-ish set of recent message ids.
type dedupCache struct {
	mu    sync.Mutex
	order []string
	set   map[string]time.Time
	max   int
}

func newDedupCache(max int) *dedupCache {
	if max < 1 {
		max = 64
	}
	return &dedupCache{set: map[string]time.Time{}, max: max}
}

func (d *dedupCache) TryAdd(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.set[id]; ok {
		return false
	}
	d.set[id] = time.Now()
	d.order = append(d.order, id)
	for len(d.order) > d.max {
		old := d.order[0]
		d.order = d.order[1:]
		delete(d.set, old)
	}
	return true
}
