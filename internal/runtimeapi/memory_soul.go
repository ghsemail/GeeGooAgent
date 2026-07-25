package runtimeapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func (h *Handler) registerMemorySoulRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/memory/soul", h.memorySoulGet)
	mux.HandleFunc("PUT /v1/memory/soul", h.memorySoulPut)
}

type soulPayload struct {
	Content string `json:"content"`
}

func (h *Handler) soulHome() string {
	return config.Home()
}

func (h *Handler) memorySoulGet(w http.ResponseWriter, r *http.Request) {
	home := h.soulHome()
	userID := resolveUserID(r)
	writeJSON(w, map[string]any{
		"content": chatprompt.LoadSoulForUser(home, userID),
		"path":    chatprompt.SoulPathForUser(home, userID),
		"user_id": userID,
		"default": false,
	})
}

func (h *Handler) memorySoulPut(w http.ResponseWriter, r *http.Request) {
	var req soulPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	home := h.soulHome()
	userID := resolveUserID(r)
	if err := chatprompt.SaveSoulForUser(home, userID, req.Content); err != nil {
		switch {
		case errors.Is(err, chatprompt.ErrSoulEmpty()):
			writeError(w, http.StatusBadRequest, "SOUL cannot be empty")
		case errors.Is(err, chatprompt.ErrSoulTooLarge()):
			writeError(w, http.StatusBadRequest, "SOUL exceeds size limit")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, map[string]any{
		"ok":      true,
		"content": chatprompt.LoadSoulForUser(home, userID),
		"path":    chatprompt.SoulPathForUser(home, userID),
		"user_id": userID,
		"message": "Saved — live next turn.",
	})
}

func soulTextForDashboard(home, userID string) string {
	return chatprompt.LoadSoulForUser(home, userID)
}
