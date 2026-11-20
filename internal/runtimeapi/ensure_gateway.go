package runtimeapi

import (
	"log/slog"
	"net/http"
)

// ensureChatGateway rebuilds the LLM gateway once when missing (e.g. catalog blip at startup).
func (h *Handler) ensureChatGateway(w http.ResponseWriter) bool {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return false
	}
	if h.App.Gateway != nil {
		return true
	}
	if err := h.App.RebuildGateway(); err != nil {
		slog.Warn("RebuildGateway after nil gateway", "error", err)
	}
	if h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return false
	}
	return true
}