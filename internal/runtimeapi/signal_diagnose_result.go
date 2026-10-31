package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func (h *Handler) registerSignalDiagnoseResultRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/signal-diagnose/result", h.signalDiagnoseResultPost)
}

type signalDiagnoseResultBody struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id,omitempty"`
	MCPToken  string `json:"mcp_token,omitempty"`
}

func (h *Handler) signalDiagnoseResultPost(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	var body signalDiagnoseResultBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
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
	ent, ok := workflowchat.LookupSignalDiagnoseResultForMonday(chat, body.RunID)
	if !ok || ent == nil {
		writeError(w, http.StatusNotFound, "signal diagnose result not found or expired")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(workflowchat.MondayAPIResponse(ent))
}
