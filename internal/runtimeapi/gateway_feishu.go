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
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

func (h *Handler) registerGatewayFeishuRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/gateway/feishu/status", h.feishuGatewayStatus)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/begin", h.feishuSetupBegin)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/poll", h.feishuSetupPoll)
	mux.HandleFunc("POST /v1/gateway/feishu/setup/manual", h.feishuSetupManual)
}

var feishuSetupMu sync.Mutex

func (h *Handler) feishuOutputDir() string {
	return h.userSettingsOutputDir()
}

func resolveMCPTokenHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get("X-MCP-Token"))
}

func (h *Handler) requireFeishuUser(w http.ResponseWriter, r *http.Request) (userID string, ok bool) {
	userID = resolveUserID(r)
	if userID == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]string{"error": "missing X-User-Id"})
		return "", false
	}
	return userID, true
}

func (h *Handler) loadOrMigrateFeishuCreds(userID string) (*feishustore.Creds, error) {
	dir := h.feishuOutputDir()
	c, err := feishustore.Load(dir, userID)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Configured() {
		return c, nil
	}
	// One-time claim of host ~/.geegoo/.env Feishu creds for this user.
	_ = config.LoadGeeGooDotEnv()
	envCfg := feishu.LoadConfigFromEnv(os.Getenv)
	if strings.TrimSpace(envCfg.AppID) == "" || strings.TrimSpace(envCfg.AppSecret) == "" {
		return c, nil
	}
	// Only migrate if no other user already owns these creds.
	list, _ := feishustore.List(dir)
	for _, existing := range list {
		if existing.AppID == envCfg.AppID {
			return c, nil
		}
	}
	migrated := &feishustore.Creds{
		UserID:       userID,
		AppID:        envCfg.AppID,
		AppSecret:    envCfg.AppSecret,
		Domain:       envCfg.Domain,
		BotName:      envCfg.BotName,
		BotOpenID:    envCfg.BotOpenID,
		AllowedUsers: append([]string(nil), envCfg.AllowedUsers...),
		HomeChannel:  envCfg.HomeChannel,
		GroupPolicy:  envCfg.GroupPolicy,
		Enabled:      true,
	}
	if err := feishustore.Save(dir, migrated); err != nil {
		return nil, err
	}
	return migrated, nil
}

func (h *Handler) syncMCPTokenFromRequest(creds *feishustore.Creds, r *http.Request) *feishustore.Creds {
	if creds == nil || !creds.Configured() {
		return creds
	}
	headerTok := resolveMCPTokenHeader(r)
	if headerTok == "" {
		return creds
	}
	if strings.TrimSpace(creds.MCPToken) == headerTok {
		return creds
	}
	creds.MCPToken = headerTok
	if err := feishustore.Save(h.feishuOutputDir(), creds); err != nil {
		return creds
	}
	return creds
}

func (h *Handler) feishuGatewayStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireFeishuUser(w, r)
	if !ok {
		return
	}
	creds, err := h.loadOrMigrateFeishuCreds(userID)
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	creds = h.syncMCPTokenFromRequest(creds, r)
	configured := creds != nil && creds.Configured()
	headerTok := resolveMCPTokenHeader(r)
	hasMCP := (creds != nil && strings.TrimSpace(creds.MCPToken) != "") || headerTok != ""

	hb, hbOK := gateway.ReadHeartbeat()
	processAlive := hbOK && gateway.HeartbeatFresh(hb, 30*time.Second)
	userConnected := false
	detail := ""
	heartbeatAt := ""
	if hbOK {
		heartbeatAt = hb.UpdatedAt.UTC().Format(time.RFC3339)
		detail = hb.Detail
		if st, ok := hb.Users[userID]; ok {
			userConnected = processAlive && st.Connected
			if st.Detail != "" {
				detail = st.Detail
			}
		} else if processAlive && configured {
			detail = "gateway process up; waiting to connect this user's bot"
		}
	}
	if !configured {
		detail = "not configured"
	}

	appID, secret, domain, botName, botOID := "", "", "feishu", "", ""
	allowed := 0
	if creds != nil {
		appID, secret = creds.AppID, creds.AppSecret
		if creds.Domain != "" {
			domain = creds.Domain
		}
		botName, botOID = creds.BotName, creds.BotOpenID
		allowed = len(creds.AllowedUsers)
	}

	writeJSON(w, map[string]any{
		"platform":          "feishu",
		"user_id":           userID,
		"configured":        configured,
		"connected":         userConnected,
		"gateway_running":   userConnected,
		"process_alive":     processAlive,
		"runtime_ok":        true,
		"detail":            detail,
		"domain":            domain,
		"connection_mode":   "websocket",
		"app_id_masked":     gateway.MaskAppID(appID),
		"app_secret_masked": gateway.MaskAppSecret(secret),
		"bot_name":          botName,
		"bot_open_id":       botOID,
		"allowed_users":     allowed,
		"has_mcp_token":     hasMCP,
		"heartbeat_at":      heartbeatAt,
		"tenant_scope":      "user",
		"store_dir":         feishustore.Dir(h.feishuOutputDir()),
		"hint":              feishuStatusHint(configured, userConnected, processAlive, hasMCP),
	})
}

func feishuStatusHint(configured, userConnected, processAlive, hasMCP bool) string {
	if !configured {
		return "扫码或手动填写 App ID / Secret；凭证按当前登录用户保存。"
	}
	if !hasMCP {
		return "飞书机器人已配置，但尚未带上 Web Gateway 的 mcp_token：请确认 Web tab 已保存 Token 后刷新本页（会自动写入飞书绑定）。"
	}
	if !processAlive {
		return "凭证已就绪，但 geegoo gateway run 未在跑：请在 Agent 机启动后自动加载你的机器人。"
	}
	if !userConnected {
		return "Gateway 进程在线，正在（或即将）连接你的飞书机器人；可稍后刷新。"
	}
	return "你的飞书 Gateway 已连接：消息会以你的 mcp_token 进入 Agent。"
}

type feishuBeginReq struct {
	Domain string `json:"domain"`
}

func (h *Handler) feishuSetupBegin(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireFeishuUser(w, r); !ok {
		return
	}
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
		"ok":            true,
		"domain":        domain,
		"device_code":   begin.DeviceCode,
		"qr_url":        begin.QRURL,
		"user_code":     begin.UserCode,
		"interval":      begin.Interval,
		"expire_in":     begin.ExpireIn,
		"qr_png_base64": qrB64,
		"expires_at":    time.Now().UTC().Add(time.Duration(begin.ExpireIn) * time.Second).Format(time.RFC3339),
	})
}

type feishuPollReq struct {
	DeviceCode string `json:"device_code"`
	Domain     string `json:"domain"`
}

func (h *Handler) feishuSetupPoll(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireFeishuUser(w, r)
	if !ok {
		return
	}
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
	doc := &feishustore.Creds{
		UserID:    userID,
		MCPToken:  resolveMCPTokenHeader(r),
		AppID:     creds.AppID,
		AppSecret: creds.AppSecret,
		Domain:    creds.Domain,
		BotName:   creds.BotName,
		BotOpenID: creds.BotOpenID,
		Enabled:   true,
	}
	if creds.OpenID != "" {
		doc.AllowedUsers = []string{creds.OpenID}
	}
	if err := feishustore.Save(h.feishuOutputDir(), doc); err != nil {
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
		"user_id":     userID,
		"tenant_scope": "user",
	})
}

type feishuManualReq struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Domain    string `json:"domain"`
}

func (h *Handler) feishuSetupManual(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireFeishuUser(w, r)
	if !ok {
		return
	}
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

	doc := &feishustore.Creds{
		UserID:    userID,
		MCPToken:  resolveMCPTokenHeader(r),
		AppID:     appID,
		AppSecret: secret,
		Domain:    domain,
		Enabled:   true,
	}
	name, oid, probeErr := feishu.ProbeBot(r.Context(), appID, secret, domain)
	if probeErr == nil {
		doc.BotName = name
		doc.BotOpenID = oid
	}
	// Preserve allowlist if re-saving.
	if prev, _ := feishustore.Load(h.feishuOutputDir(), userID); prev != nil {
		doc.AllowedUsers = prev.AllowedUsers
		if doc.MCPToken == "" {
			doc.MCPToken = prev.MCPToken
		}
	}
	if err := feishustore.Save(h.feishuOutputDir(), doc); err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{
		"ok":           true,
		"status":       "completed",
		"app_id":       appID,
		"domain":       domain,
		"bot_name":     doc.BotName,
		"bot_open_id":  doc.BotOpenID,
		"user_id":      userID,
		"tenant_scope": "user",
		"verified":     probeErr == nil,
	})
}
