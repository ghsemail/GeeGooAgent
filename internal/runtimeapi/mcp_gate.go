package runtimeapi

import (
	"net/http"
	"strings"
)

func userProvidedMCPToken(r *http.Request, bodyToken string) string {
	if r == nil {
		return strings.TrimSpace(bodyToken)
	}
	if v := strings.TrimSpace(r.Header.Get("X-MCP-Token")); v != "" {
		return v
	}
	return strings.TrimSpace(bodyToken)
}

// resolveChatMCPToken prefers the caller's per-user token (X-MCP-Token / body) so
// multi-tenant BFF chat uses the acting user's GeeGoo identity. Runtime config
// mcp_token is only a single-user fallback (CLI / dev).
func resolveChatMCPToken(r *http.Request, bodyToken, configToken string) string {
	if v := userProvidedMCPToken(r, bodyToken); v != "" {
		return v
	}
	return strings.TrimSpace(configToken)
}

// requireMCPTokenForChat allows chat when runtime config or the caller supplies a token.
func requireMCPTokenForChat(w http.ResponseWriter, r *http.Request, bodyToken, configToken string) bool {
	if resolveChatMCPToken(r, bodyToken, configToken) != "" {
		return true
	}
	writeError(w, http.StatusUnauthorized, "missing mcp_token: set mcp_token in agent-runtime config")
	return false
}

func (h *Handler) configMCPToken() string {
	if h == nil || h.App == nil || h.App.Config == nil {
		return ""
	}
	return h.App.Config.MCPToken()
}
