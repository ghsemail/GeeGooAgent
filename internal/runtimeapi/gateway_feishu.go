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
	appID := cfg.AppID
	masked := appID
	if len(masked) > 8 {
		masked = masked[:4] + "…" + masked[len(masked)-4:]
	}
	writeJSON(w, map[string]any{
		"platform":       "feishu",
		"configured":     st.Configured,
		"connected":      st.Connected,
		"detail":         st.Detail,
		"domain":         cfg.Domain,
		"connection_mode": cfg.ConnectionMode,
		"app_id_masked":  masked,
		"bot_name":       cfg.BotName,
		"bot_open_id":    cfg.BotOpenID,
		"allowed_users":  len(cfg.AllowedUsers),
		"home_channel":   cfg.HomeChannel,
		"group_policy":   cfg.GroupPolicy,
		"env_file":       config.EnvFilePath(),
		"hint":           "Run `geegoo gateway run` on the Agent host to keep the WebSocket connected.",
	})
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
