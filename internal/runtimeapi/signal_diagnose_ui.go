package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func (h *Handler) registerSignalDiagnoseUIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/signal-diagnose/ui-payload", h.signalDiagnoseUIPayloadGet)
	mux.HandleFunc("POST /v1/signal-diagnose/ui-payload", h.signalDiagnoseUIPayloadPost)
}

type signalDiagnoseUIPayloadBody struct {
	SessionID string `json:"session_id"`
	Key       string `json:"diagnose_ui_key"`
	MCPToken  string `json:"mcp_token"`
}

func (h *Handler) signalDiagnoseUIPayloadGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("diagnose_ui_key"))
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("key"))
	}
	body := signalDiagnoseUIPayloadBody{
		SessionID: strings.TrimSpace(r.URL.Query().Get("session_id")),
		Key:       key,
		MCPToken:  strings.TrimSpace(r.URL.Query().Get("mcp_token")),
	}
	h.serveSignalDiagnoseUIPayload(w, r, body)
}

func (h *Handler) signalDiagnoseUIPayloadPost(w http.ResponseWriter, r *http.Request) {
	var body signalDiagnoseUIPayloadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	h.serveSignalDiagnoseUIPayload(w, r, body)
}

func (h *Handler) serveSignalDiagnoseUIPayload(w http.ResponseWriter, r *http.Request, body signalDiagnoseUIPayloadBody) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if !requireMCPTokenForChat(w, r, body.MCPToken, h.configMCPToken()) {
		return
	}
	sessionID := strings.TrimSpace(body.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required")
		return
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		writeError(w, http.StatusBadRequest, "diagnose_ui_key required")
		return
	}
	chat, err := store.Load(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	userID := resolveUserID(r)
	if !enforceSessionAccess(w, chat, userID) {
		return
	}
	ent, ok := workflowchat.LookupSignalDiagnoseUIPayload(chat, key)
	if !ok {
		writeError(w, http.StatusNotFound, "diagnose ui payload not found or expired")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"diagnose_ui_key": key,
		"session_id":      sessionID,
		"run_id":          ent.RunID,
		"stock_code":      ent.StockCode,
		"chart_probe":     ent.ChartProbe,
		"signal_eval":     ent.SignalEval,
		"created_at":      ent.CreatedAt,
		"expires_at":      ent.ExpiresAt,
	})
}
