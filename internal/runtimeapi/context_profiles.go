package runtimeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func (h *Handler) registerContextProfileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/context/profiles", h.contextProfilesGet)
	mux.HandleFunc("GET /v1/context/profiles/inspect", h.contextProfilesInspect)
	mux.HandleFunc("PUT /v1/context/profiles/{kind}/{key}", h.contextProfilePut)
}

type contextProfilePayload struct {
	Content string `json:"content"`
}

func (h *Handler) profileLimits() chatprompt.ProfileLimits {
	limits := chatprompt.DefaultProfileLimits()
	if h.App != nil && h.App.Config != nil {
		limits.MaxMergedBytes = h.App.Config.EffectiveContextProfileMaxMergedBytes()
		limits.MaxProfilesPerSession = h.App.Config.EffectiveContextProfileMaxPerSession()
	}
	return limits
}

func (h *Handler) contextProfilesGet(w http.ResponseWriter, r *http.Request) {
	home := config.Home()
	userID := resolveUserID(r)
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if kind != "" {
		var ref chatprompt.ProfileRef
		switch chatprompt.ProfileKind(kind) {
		case chatprompt.ProfileGlobal, chatprompt.ProfileUserDefault:
			ref = chatprompt.ProfileRef{Kind: chatprompt.ProfileKind(kind), Key: key}
		default:
			var err error
			ref, err = chatprompt.ParseProfileRef(kind + ":" + key)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		lp, ok := chatprompt.LoadProfile(home, userID, ref)
		writeJSON(w, map[string]any{
			"ref":     ref.String(),
			"path":    lp.Path,
			"content": lp.Content,
			"exists":  ok,
			"bytes":   lp.Bytes,
			"user_id": userID,
		})
		return
	}
	sessionRefs := parseProfileRefQuery(r.URL.Query().Get("refs"))
	if sid := strings.TrimSpace(r.URL.Query().Get("session_id")); sid != "" {
		if store, err := h.App.SessionStore(); err == nil {
			if chat, err := store.Load(sid); err == nil && chat != nil {
				sessionRefs = chatsession.MergeContextProfiles(sessionRefs, chatsession.ContextProfilesFromSession(chat))
			}
		}
	}
	merge := chatprompt.InspectProfiles(home, userID, sessionRefs, h.profileLimits())
	writeJSON(w, map[string]any{
		"user_id":  userID,
		"refs":     sessionRefs,
		"merged":   merge.Text,
		"profiles": merge.Profiles,
		"truncated": merge.Truncated,
		"limits": map[string]any{
			"max_merged_bytes":         h.profileLimits().MaxMergedBytes,
			"max_profiles_per_session": h.profileLimits().MaxProfilesPerSession,
		},
	})
}

func (h *Handler) contextProfilesInspect(w http.ResponseWriter, r *http.Request) {
	h.contextProfilesGet(w, r)
}

func (h *Handler) contextProfilePut(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.PathValue("kind"))
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "_" {
		key = ""
	}
	ref := chatprompt.ProfileRef{Kind: chatprompt.ProfileKind(kind), Key: key}
	if ref.Kind == chatprompt.ProfileGlobal || ref.Kind == chatprompt.ProfileUserDefault {
		// key optional
	} else if key == "" {
		writeError(w, http.StatusBadRequest, "profile key required")
		return
	}
	var req contextProfilePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	home := config.Home()
	userID := resolveUserID(r)
	if err := chatprompt.SaveProfile(home, userID, ref, req.Content); err != nil {
		switch {
		case errors.Is(err, chatprompt.ErrAgentsEmpty()):
			writeError(w, http.StatusBadRequest, "AGENTS cannot be empty")
		case errors.Is(err, chatprompt.ErrAgentsTooLarge()):
			writeError(w, http.StatusBadRequest, "AGENTS exceeds size limit")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	lp, _ := chatprompt.LoadProfile(home, userID, ref)
	writeJSON(w, map[string]any{
		"ok":      true,
		"ref":     ref.String(),
		"path":    lp.Path,
		"content": lp.Content,
		"bytes":   lp.Bytes,
		"message": "Saved — live next turn.",
	})
}

func parseProfileRefQuery(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
