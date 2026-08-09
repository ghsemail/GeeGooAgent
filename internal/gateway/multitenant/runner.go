package multitenant

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/gateway"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

// Runner loads per-user Feishu bots and keeps them connected with hot reload.
type Runner struct {
	App      *app.App
	Sessions *gateway.SessionMap
	StoreDir string
	DryRun   bool

	mu      sync.Mutex
	tenants map[string]*tenantSlot
	agentMu sync.Mutex
	chatMu  map[string]*sync.Mutex
	seen    *dedupCache
}

type tenantSlot struct {
	creds   feishustore.Creds
	fp      string
	adapter *feishu.Adapter
	cancel  context.CancelFunc
}

// NewRunner builds a multi-user Feishu gateway.
func NewRunner(application *app.App, sessions *gateway.SessionMap, storeDir string, dryRun bool) *Runner {
	return &Runner{
		App:      application,
		Sessions: sessions,
		StoreDir: storeDir,
		DryRun:   dryRun,
		tenants:  map[string]*tenantSlot{},
		chatMu:   map[string]*sync.Mutex{},
		seen:     newDedupCache(4096),
	}
}

// Start blocks until ctx is cancelled.
func (tr *Runner) Start(ctx context.Context) error {
	if tr == nil || tr.App == nil {
		return errString("nil tenant runner")
	}
	tr.syncTenants(ctx)
	go tr.heartbeatLoop(ctx)
	t := time.NewTicker(8 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			tr.stopAll()
			return nil
		case <-t.C:
			tr.syncTenants(ctx)
		}
	}
}

func (tr *Runner) stopAll() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	for id, slot := range tr.tenants {
		tr.stopSlotLocked(id, slot)
	}
}

func (tr *Runner) stopSlotLocked(id string, slot *tenantSlot) {
	if slot == nil {
		return
	}
	if slot.cancel != nil {
		slot.cancel()
	}
	if slot.adapter != nil {
		_ = slot.adapter.Disconnect(context.Background())
	}
	delete(tr.tenants, id)
}

func (tr *Runner) syncTenants(ctx context.Context) {
	list, err := feishustore.List(tr.StoreDir)
	if err != nil {
		slog.Warn("gateway: list feishu creds", "err", err)
		return
	}
	wanted := map[string]feishustore.Creds{}
	for _, c := range list {
		if !c.Enabled || !c.Configured() {
			continue
		}
		wanted[c.UserID] = c
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	for id, slot := range tr.tenants {
		c, ok := wanted[id]
		if !ok || c.Fingerprint() != slot.fp {
			slog.Info("gateway: stopping tenant", "user_id", id)
			tr.stopSlotLocked(id, slot)
		}
	}
	for id, c := range wanted {
		if _, ok := tr.tenants[id]; ok {
			continue
		}
		tr.startSlotLocked(ctx, c)
	}
	if len(tr.tenants) == 0 {
		slog.Info("gateway: no per-user Feishu bots configured yet")
	}
}

func (tr *Runner) startSlotLocked(ctx context.Context, c feishustore.Creds) {
	cfg := credsToFeishuConfig(c)
	ad := feishu.NewAdapter(cfg)
	if !ad.Configured() {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	slot := &tenantSlot{creds: c, fp: c.Fingerprint(), adapter: ad, cancel: cancel}
	tr.tenants[c.UserID] = slot
	ownerID := c.UserID
	mcp := c.MCPToken
	adapter := ad
	allow := cfg.AllowedSet()
	allowAll := cfg.GatewayAllowAll()
	toolProgress := cfg.ToolProgress

	go func() {
		slog.Info("gateway: connecting tenant feishu", "user_id", ownerID, "app", gateway.MaskAppID(c.AppID))
		err := adapter.Connect(runCtx, func(cctx context.Context, ev gateway.InboundEvent) error {
			return tr.handleOwned(cctx, ev, ownedInbound{
				OwnerUserID:  ownerID,
				MCPToken:     mcp,
				Adapter:      adapter,
				Allowed:      allow,
				AllowAll:     allowAll,
				ToolProgress: toolProgress,
			})
		})
		if err != nil && runCtx.Err() == nil {
			slog.Error("gateway: tenant feishu stopped", "user_id", ownerID, "err", err)
		}
	}()
}

func (tr *Runner) heartbeatLoop(ctx context.Context) {
	write := func() {
		tr.mu.Lock()
		users := map[string]gateway.UserHeartbeatStatus{}
		anyConnected := false
		for id, slot := range tr.tenants {
			st := slot.adapter.Status()
			users[id] = gateway.UserHeartbeatStatus{
				Connected:  st.Connected,
				Configured: st.Configured,
				BotName:    slot.creds.BotName,
				AppIDMask:  gateway.MaskAppID(slot.creds.AppID),
				Detail:     st.Detail,
			}
			if st.Connected {
				anyConnected = true
			}
		}
		n := len(tr.tenants)
		tr.mu.Unlock()
		_ = gateway.WriteHeartbeat(gateway.HeartbeatSnapshot{
			Platform:   "feishu",
			Connected:  anyConnected,
			Configured: n > 0,
			Detail:     fmtTenantDetail(n, anyConnected),
			Users:      users,
		})
	}
	write()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = gateway.WriteHeartbeat(gateway.HeartbeatSnapshot{
				Platform:  "feishu",
				Connected: false,
				Detail:    "gateway stopped",
				Users:     map[string]gateway.UserHeartbeatStatus{},
			})
			return
		case <-t.C:
			write()
		}
	}
}

func fmtTenantDetail(n int, connected bool) string {
	if n == 0 {
		return "no tenants"
	}
	if connected {
		return "multi-tenant feishu connected"
	}
	return "multi-tenant feishu connecting"
}

func credsToFeishuConfig(c feishustore.Creds) feishu.Config {
	domain := c.Domain
	if domain == "" {
		domain = "feishu"
	}
	policy := c.GroupPolicy
	if policy == "" {
		policy = "allowlist"
	}
	return feishu.Config{
		AppID:          c.AppID,
		AppSecret:      c.AppSecret,
		Domain:         domain,
		ConnectionMode: "websocket",
		AllowedUsers:   append([]string(nil), c.AllowedUsers...),
		AllowAllUsers:  len(c.AllowedUsers) == 0,
		HomeChannel:    c.HomeChannel,
		GroupPolicy:    policy,
		RequireMention: true,
		BotOpenID:      c.BotOpenID,
		BotName:        c.BotName,
		ToolProgress:   true,
	}
}

type ownedInbound struct {
	OwnerUserID  string
	MCPToken     string
	Adapter      gateway.PlatformAdapter
	Allowed      map[string]struct{}
	AllowAll     bool
	ToolProgress bool
}

func (tr *Runner) lockFor(key string) *sync.Mutex {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if m, ok := tr.chatMu[key]; ok {
		return m
	}
	m := &sync.Mutex{}
	tr.chatMu[key] = m
	return m
}

type errString string

func (e errString) Error() string { return string(e) }

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
