package runtimeapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

func (h *Handler) registerGatewayFeishuRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/gateway/feishu/status", h.feishuGatewayStatus)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/begin", h.feishuSetupBegin)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/poll", h.feishuSetupPoll)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/manual", h.feishuSetupManual)
}

var feishuSetupMu sync.Mutex

func (h *Handler) feishuGatewayStatus(w http.ResponseWriter, r *http.Request) {
	_ = config.LoadGeeGooDotEnv()
	cfg := feishu.LoadConfigFromEnv(os.Getenv)
	ad := feishu.NewAdapter(cfg)
	st := ad.Status()

	hb, hbOK := gateway.ReadHeartbeat()
	gatewayRunning := hbOK && gateway.HeartbeatFresh(hb, 30*time.Second) && hb.Connected
	connected := gatewayRunning
	detail := st.Detail
	heartbeatAt := ""
	if hbOK {
		heartbeatAt = hb.UpdatedAt.UTC().Format(time.RFC3339)
		if hb.Detail != "" {
			detail = hb.Detail
		}
		if !gatewayRunning && gateway.HeartbeatFresh(hb, 30*time.Second) && !hb.Connected {
			detail = hb.Detail
		}
	}

	writeJSON(w, map[string]any{
		"platform":           "feishu",
		"configured":         st.Configured,
		"connected":          connected,
		"gateway_running":    gatewayRunning,
		"runtime_ok":         true,
		"detail":             detail,
		"domain":             cfg.Domain,
		"connection_mode":    cfg.ConnectionMode,
		"app_id_masked":      gateway.MaskAppID(cfg.AppID),
		"app_secret_masked":  gateway.MaskAppSecret(cfg.AppSecret),
		"bot_name":           cfg.BotName,
		"bot_open_id":        cfg.BotOpenID,
		"allowed_users":      len(cfg.AllowedUsers),
		"home_channel":       cfg.HomeChannel,
		"group_policy":       cfg.GroupPolicy,
		"env_file":           config.EnvFilePath(),
		"heartbeat_at":       heartbeatAt,
		"tenant_scope":       "host", // shared ~/.geegoo/.env on Agent host; not per Dashboard user
		"hint":               feishuStatusHint(st.Configured, gatewayRunning),
	})
}

func feishuStatusHint(configured, gatewayRunning bool) string {
	if !configured {
		return "扫码或手动填写 App ID / Secret 后，在 Agent 机运行 geegoo gateway run。"
	}
	if !gatewayRunning {
		return "凭证已就绪，但 IM Gateway 未在跑：请在 Agent 机执行 geegoo gateway run。"
	}
	return "IM Gateway 运行中：飞书消息会进入 Agent。"
}

type feishuBeginReq struct {
	Domain string `json:"domain"`
}

func (h *Handler) feishuSetupBegin(w http.ResponseWriter, r *http.Request) {
	var req feishuBeginReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = "feishu"
	}
	if domain != "feishu" && domain != "lark" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "domain must be feishu or lark"})
		return
	}

	ctx := r.Context()
	if err := feishu.InitRegistration(ctx, domain); err != nil {
		writeJSONStatus(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	begin, err := feishu.BeginRegistration(ctx, domain)
	if err != nil {
		writeJSONStatus(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	qrPNG, err := qrcode.Encode(begin.QRURL, qrcode.Medium, 256)
	qrB64 := ""
	if err == nil {
		qrB64 = base64.StdEncoding.EncodeToString(qrPNG)
	}

	writeJSON(w, map[string]any{
		"ok":           true,
		"domain":       domain,
		"device_code":  begin.DeviceCode,
		"qr_url":       begin.QRURL,
		"user_code":    begin.UserCode,
		"interval":     begin.Interval,
		"expire_in":    begin.ExpireIn,
		"qr_png_base64": qrB64,
		"expires_at":   time.Now().UTC().Add(time.Duration(begin.ExpireIn) * time.Second).Format(time.RFC3339),
	})
}

type feishuPollReq struct {
	DeviceCode string `json:"device_code"`
	Domain     string `json:"domain"`
	Interval   int    `json:"interval"`
	ExpireIn   int    `json:"expire_in"`
}

func (h *Handler) feishuSetupPoll(w http.ResponseWriter, r *http.Request) {
	var req feishuPollReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.DeviceCode) == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "device_code required"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = "feishu"
	}

	feishuSetupMu.Lock()
	defer feishuSetupMu.Unlock()

	creds, status, err := feishu.PollOnce(r.Context(), req.DeviceCode, domain)
	if status == "pending" {
		writeJSON(w, map[string]any{"ok": true, "status": "pending"})
		return
	}
	if status == "failed" || err != nil {
		msg := "registration failed"
		if err != nil {
			msg = err.Error()
		}
		writeJSONStatus(w, http.StatusConflict, map[string]any{"ok": false, "status": "failed", "error": msg})
		return
	}
	if creds == nil {
		writeJSON(w, map[string]any{"ok": true, "status": "pending"})
		return
	}

	name, oid, _ := feishu.ProbeBot(r.Context(), creds.AppID, creds.AppSecret, creds.Domain)
	if name != "" {
		creds.BotName = name
	}
	if oid != "" {
		creds.BotOpenID = oid
	}
	extra := map[string]string{}
	if creds.BotOpenID != "" {
		extra["FEISHU_BOT_OPEN_ID"] = creds.BotOpenID
	}
	if creds.BotName != "" {
		extra["FEISHU_BOT_NAME"] = creds.BotName
	}
	if creds.OpenID != "" {
		extra["FEISHU_ALLOWED_USERS"] = creds.OpenID
	}
	if err := config.SaveFeishuEnv(creds.AppID, creds.AppSecret, creds.Domain, extra); err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{
		"ok":          true,
		"status":      "completed",
		"app_id":      creds.AppID,
		"domain":      creds.Domain,
		"bot_name":    creds.BotName,
		"bot_open_id": creds.BotOpenID,
		"open_id":     creds.OpenID,
		"env_file":    config.EnvFilePath(),
	})
}

type feishuManualReq struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Domain    string `json:"domain"`
}

func (h *Handler) feishuSetupManual(w http.ResponseWriter, r *http.Request) {
	var req feishuManualReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	appID := strings.TrimSpace(req.AppID)
	secret := strings.TrimSpace(req.AppSecret)
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = "feishu"
	}
	if appID == "" || secret == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "app_id and app_secret required"})
		return
	}

	extra := map[string]string{}
	name, oid, err := feishu.ProbeBot(r.Context(), appID, secret, domain)
	if err == nil {
		if name != "" {
			extra["FEISHU_BOT_NAME"] = name
		}
		if oid != "" {
			extra["FEISHU_BOT_OPEN_ID"] = oid
		}
	}
	if err := config.SaveFeishuEnv(appID, secret, domain, extra); err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{
		"ok":          true,
		"status":      "completed",
		"app_id":      appID,
		"domain":      domain,
		"bot_name":    extra["FEISHU_BOT_NAME"],
		"bot_open_id": extra["FEISHU_BOT_OPEN_ID"],
		"env_file":    config.EnvFilePath(),
		"verified":    err == nil,
	})
}
