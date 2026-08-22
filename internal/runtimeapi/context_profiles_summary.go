package runtimeapi

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func (h *Handler) buildContextProfilesSummary(userID string) map[string]any {
	home := config.Home()
	merge := chatprompt.InspectProfiles(home, userID, nil, h.profileLimits())
	loaded := 0
	paths := make([]string, 0, len(merge.Profiles))
	for _, p := range merge.Profiles {
		if !p.Missing && p.Bytes > 0 {
			loaded++
			paths = append(paths, p.Path)
		}
	}
	dbLoaded := 0
	if h.App != nil && h.App.Preferences != nil {
		if n, err := h.App.Preferences.CountLoaded(context.Background(), userID); err == nil {
			dbLoaded = n
		}
	}
	if dbLoaded > loaded {
		loaded = dbLoaded
	}
	return map[string]any{
		"loaded_count": loaded,
		"db_scopes":    dbLoaded,
		"merged_bytes": len([]byte(merge.Text)),
		"truncated":    merge.Truncated,
		"paths":        paths,
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
