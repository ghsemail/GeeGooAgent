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
	writeJSON(w, map[string]any{
		"content": chatprompt.LoadSoulFromHome(home),
		"path":    chatprompt.SoulPath(home),
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
	if err := chatprompt.SaveSoulToHome(home, req.Content); err != nil {
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
		"content": chatprompt.LoadSoulFromHome(home),
		"path":    chatprompt.SoulPath(home),
		"message": "Saved — live next turn.",
	})
}

func soulTextForDashboard(home string) string {
	return chatprompt.LoadSoulFromHome(home)
}
