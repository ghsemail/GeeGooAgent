package runtimeapi

import (
	"context"
	"net/http"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory/scoped"
)

func (h *Handler) buildSessionContextInjection(session *chatsession.ChatSession) map[string]any {
	if session == nil {
		return nil
	}
	scopes := chatsession.ActiveScopesFromSession(session)
	userID := chatsession.UserIDFromSession(session)
	status := h.profileStatusEntries(userID, scopes)
	mergedBytes := 0
	for _, p := range status {
		if n, ok := p["bytes"].(int); ok {
			mergedBytes += n
		}
	}
	return map[string]any{
		"active_scopes": scopes,
		"merged_bytes":  mergedBytes,
		"truncated":     false,
		"profiles":      status,
		"assemble_order": []string{
			"SOUL",
			"global AGENTS",
			"user AGENTS",
			"session scopes (market/stock/automation)",
			"hard rules",
			"retrieval gate (facts/episodes)",
			"procedural skills",
		},
	}
}

func (h *Handler) lightweightProfilesInspect(userID string, sessionRefs []string) map[string]any {
	status := h.profileStatusEntries(userID, sessionRefs)
	mergedBytes := 0
	for _, p := range status {
		if n, ok := p["bytes"].(int); ok {
			mergedBytes += n
		}
	}
	return map[string]any{
		"user_id":   userID,
		"refs":      sessionRefs,
		"merged":    "",
		"profiles":  status,
		"truncated": false,
		"limits": map[string]any{
			"max_merged_bytes":         h.profileLimits().MaxMergedBytes,
			"max_profiles_per_session": h.profileLimits().MaxProfilesPerSession,
		},
	}
}

func (h *Handler) profileStatusEntries(userID string, sessionScopes []string) []map[string]any {
	home := config.Home()
	limits := h.profileLimits()
	refs := []chatprompt.ProfileRef{
		{Kind: chatprompt.ProfileGlobal},
		{Kind: chatprompt.ProfileUserDefault, Key: chatprompt.SanitizeUserID(userID)},
	}
	if resolved, err := chatprompt.ResolveProfileRefs(sessionScopes, limits); err == nil {
		refs = append(refs, resolved...)
	}

	dbScopes := map[string]scoped.PreferenceRow{}
	if h.App != nil && h.App.Preferences != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		rows, err := h.App.Preferences.ListSummary(ctx, userID)
		cancel()
		if err == nil {
			for _, row := range rows {
				dbScopes[row.Scope] = row
			}
		}
	}

	out := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		scope := scoped.ScopeFromRef(ref)
		entry := map[string]any{
			"ref":     ref.String(),
			"scope":   scope,
			"path":    chatprompt.AgentsPathForRef(home, userID, ref),
			"missing": true,
		}
		if row, ok := dbScopes[scope]; ok && row.Bytes > 0 {
			entry["bytes"] = row.Bytes
			entry["source"] = row.Source
			entry["missing"] = false
			entry["updated_at"] = row.UpdatedAt.Format(time.RFC3339)
			out = append(out, entry)
			continue
		}
		lp, ok := chatprompt.LoadProfileFromFile(home, userID, ref)
		if ok && lp.Bytes > 0 {
			entry["bytes"] = lp.Bytes
			entry["source"] = "file"
			entry["missing"] = false
			out = append(out, entry)
			continue
		}
		out = append(out, entry)
	}
	return out
}

func (h *Handler) scopedPreferencesSummary(userID string) map[string]any {
	loaded := 0
	dbScopes := 0
	var prefs []map[string]any
	if h.App != nil && h.App.Preferences != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		rows, err := h.App.Preferences.ListSummary(ctx, userID)
		cancel()
		if err == nil {
			dbScopes = len(rows)
			prefs = make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				if row.Bytes <= 0 {
					continue
				}
				loaded++
				prefs = append(prefs, map[string]any{
					"scope":      row.Scope,
					"bytes":      row.Bytes,
					"source":     row.Source,
					"updated_at": row.UpdatedAt.Format(time.RFC3339),
				})
			}
		}
	}
	return map[string]any{
		"loaded_count": loaded,
		"db_scopes":    dbScopes,
		"preferences":  prefs,
		"inspect_url":  "/v1/context/profiles/inspect",
		"kinds": []string{
			string(chatprompt.ProfileGlobal),
			string(chatprompt.ProfileUserDefault),
			string(chatprompt.ProfileMarket),
			string(chatprompt.ProfileStock),
			string(chatprompt.ProfileAutomation),
		},
	}
}

func (h *Handler) memoryScopedPreferencesList(w http.ResponseWriter, r *http.Request) {
	userID := resolveUserID(r)
	writeJSON(w, h.scopedPreferencesSummary(userID))
}
