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
	if h.App != nil {
		if store, err := h.App.SessionStore(); err == nil && store != nil {
			chat, err := store.Load(sessionID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if chat == nil {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			if !enforceSessionAccess(w, chat, resolveUserID(r)) {
				return
			}
		}
	}
	answer := strings.TrimSpace(req.Answer)
	if !req.Skip && answer == "" {
		writeError(w, http.StatusBadRequest, "answer required unless skip=true")
		return
	}
	ok := !req.Skip
	if !h.clarify.Answer(sessionID, answer, ok) {
		writeError(w, http.StatusNotFound, "no pending clarify for session")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func (h *Handler) clarifyFn(fallback context.Context, sessionID string, onPending func(PendingClarify)) func(context.Context, string, []string) (string, bool) {
	return func(waitCtx context.Context, question string, choices []string) (string, bool) {
		ctx := waitCtx
		if ctx == nil {
			ctx = fallback
		}
		return h.clarify.Wait(ctx, sessionID, question, choices, onPending)
	}
}
