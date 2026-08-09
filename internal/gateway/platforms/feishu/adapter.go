package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

// Adapter implements gateway.PlatformAdapter for Feishu/Lark (M1: WebSocket text).
type Adapter struct {
	cfg Config

	api    *lark.Client
	ws     *larkws.Client
	botOID string
	botName string

	connected atomic.Bool
	mu        sync.Mutex
	cancel    context.CancelFunc
}

// NewAdapter builds a Feishu adapter. Returns nil-capable Configured()=false when creds missing.
func NewAdapter(cfg Config) *Adapter {
	return &Adapter{cfg: cfg}
}

func (a *Adapter) Platform() gateway.Platform { return gateway.PlatformFeishu }

func (a *Adapter) Configured() bool {
	return a != nil && strings.TrimSpace(a.cfg.AppID) != "" && strings.TrimSpace(a.cfg.AppSecret) != ""
}

func (a *Adapter) Status() gateway.AdapterStatus {
	detail := "not configured"
	if a.Configured() {
		detail = "mode=" + a.cfg.ConnectionMode + " domain=" + a.cfg.Domain
		if a.connected.Load() {
			detail += " connected"
		} else {
			detail += " disconnected"
		}
		if a.botOID != "" {
			detail += " bot_open_id=" + a.botOID
		}
	}
	return gateway.AdapterStatus{
		Platform:   gateway.PlatformFeishu,
		Configured: a.Configured(),
		Connected:  a.connected.Load(),
		Detail:     detail,
	}
}

// Connect starts the WebSocket long connection and blocks until ctx is cancelled.
func (a *Adapter) Connect(ctx context.Context, onInbound gateway.InboundHandler) error {
	if !a.Configured() {
		return fmt.Errorf("feishu: app id/secret not set")
	}
	if strings.ToLower(a.cfg.ConnectionMode) != "websocket" {
		return fmt.Errorf("feishu: connection mode %q not supported in M1 (use websocket)", a.cfg.ConnectionMode)
	}
	if len(a.cfg.AllowedUsers) == 0 && !a.cfg.AllowAllUsers {
		slog.Warn("feishu: FEISHU_ALLOWED_USERS empty; accepting all users (set allowlist for production)")
	}

	baseURL := lark.FeishuBaseUrl
	if strings.EqualFold(a.cfg.Domain, "lark") {
		baseURL = lark.LarkBaseUrl
	}
	a.api = lark.NewClient(a.cfg.AppID, a.cfg.AppSecret, lark.WithOpenBaseUrl(baseURL))

	a.botOID = a.cfg.BotOpenID
	a.botName = a.cfg.BotName
	if a.botOID == "" {
		if oid, name, err := a.fetchBotIdentity(ctx); err != nil {
			slog.Warn("feishu: bot identity auto-detect failed", "err", err)
		} else {
			a.botOID = oid
			if a.botName == "" {
				a.botName = name
			}
			slog.Info("feishu: bot identity", "open_id", a.botOID, "name", a.botName)
		}
	}

	allowed := a.cfg.AllowedSet()
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			ev, ok := NormalizeMessage(event, a.botOID, a.botName, a.cfg.RequireMention, a.cfg.GroupPolicy, allowed)
			if !ok {
				return nil
			}
			if onInbound == nil {
				return nil
			}
			if err := onInbound(ctx, ev); err != nil {
				slog.Error("feishu: inbound handler", "err", err)
			}
			return nil
		})

	opts := []larkws.ClientOption{
		larkws.WithEventHandler(handler),
		larkws.WithDomain(baseURL),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	}
	a.ws = larkws.NewClient(a.cfg.AppID, a.cfg.AppSecret, opts...)

	runCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("feishu: websocket starting", "domain", a.cfg.Domain)
		a.connected.Store(true)
		err := a.ws.Start(runCtx)
		a.connected.Store(false)
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.ws.Close()
		cancel()
		return nil
	case err := <-errCh:
		cancel()
		return err
	}
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	_ = ctx
	a.mu.Lock()
	cancel := a.cancel
	a.cancel = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if a.ws != nil {
		a.ws.Close()
	}
	a.connected.Store(false)
	return nil
}

func (a *Adapter) SendText(ctx context.Context, msg gateway.OutboundText) error {
	if a.api == nil {
		baseURL := lark.FeishuBaseUrl
		if strings.EqualFold(a.cfg.Domain, "lark") {
			baseURL = lark.LarkBaseUrl
		}
		a.api = lark.NewClient(a.cfg.AppID, a.cfg.AppSecret, lark.WithOpenBaseUrl(baseURL))
	}
	content, _ := json.Marshal(map[string]string{"text": msg.Text})
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.CreateMessageV1ReceiveIDTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType("text").
			Content(string(content)).
			Build()).
		Build()
	resp, err := a.api.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("feishu send: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (a *Adapter) SendMedia(ctx context.Context, chatID string, filename string, data []byte, mime string) error {
	_ = ctx
	_ = chatID
	_ = filename
	_ = data
	_ = mime
	return gateway.ErrNotImplemented{Feature: "feishu SendMedia (M2)"}
}

func (a *Adapter) fetchBotIdentity(ctx context.Context) (openID, name string, err error) {
	resp, err := a.api.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return "", "", err
	}
	if resp == nil || resp.StatusCode != 200 {
		return "", "", fmt.Errorf("bot/v3/info status=%v", resp)
	}
	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID  string `json:"open_id"`
			AppName string `json:"app_name"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		return "", "", err
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("bot/v3/info code=%d", result.Code)
	}
	return result.Bot.OpenID, result.Bot.AppName, nil
}
