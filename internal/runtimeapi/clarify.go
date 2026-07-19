package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type clarifyRequest struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
	Skip      bool   `json:"skip"`
}

func (h *Handler) registerClarifyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/clarify", h.chatClarify)
}

func (h *Handler) chatClarify(w http.ResponseWriter, r *http.Request) {
	if h.clarify == nil {
		writeError(w, http.StatusServiceUnavailable, "clarify hub not configured")
		return
	}
	var req clarifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required")
		return
	}
	answer := strings.TrimSpace(req.Answer)
	ok := !req.Skip
	if !ok && answer == "" {
		writeError(w, http.StatusBadRequest, "answer required unless skip=true")
		return
	}
	if !h.clarify.Answer(sessionID, answer, ok) {
		writeError(w, http.StatusNotFound, "no pending clarify for session")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func (h *Handler) clarifyFn(ctx context.Context, sessionID string, onPending func(PendingClarify)) func(string, []string) (string, bool) {
	return func(question string, choices []string) (string, bool) {
		return h.clarify.Wait(ctx, sessionID, question, choices, onPending)
	}
}
