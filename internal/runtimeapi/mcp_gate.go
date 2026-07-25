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

// requireUserMCPTokenForChat rejects multi-tenant chat when the caller did not supply a user MCP token.
// CLI / single-tenant calls without X-User-Id may still use config fallback via resolveMCPToken.
func requireUserMCPTokenForChat(w http.ResponseWriter, r *http.Request, bodyToken string) bool {
	if resolveUserID(r) == "" {
		return true
	}
	if userProvidedMCPToken(r, bodyToken) != "" {
		return true
	}
	writeError(w, http.StatusUnauthorized, "missing mcp_token: generate one in profile center")
	return false
}
