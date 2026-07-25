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

// resolveChatMCPToken prefers agent-runtime config mcp_token (trading user identity
// for MCP tools) over caller-supplied tokens. Ops portals may send an admin
// mcp_token for BFF auth; tool calls must still use the runtime config token.
func resolveChatMCPToken(r *http.Request, bodyToken, configToken string) string {
	if v := strings.TrimSpace(configToken); v != "" {
		return v
	}
	return userProvidedMCPToken(r, bodyToken)
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
