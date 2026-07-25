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

// requireUserMCPTokenForChat rejects interactive HTTP chat when the caller did not supply a user MCP token.
// Config-file fallback must not be used for web/BFF traffic — each operator needs their own token.
func requireUserMCPTokenForChat(w http.ResponseWriter, r *http.Request, bodyToken string) bool {
	if userProvidedMCPToken(r, bodyToken) != "" {
		return true
	}
	writeError(w, http.StatusUnauthorized, "missing mcp_token: generate one in profile center")
	return false
}

// resolveInteractiveMCPToken returns only caller-supplied tokens (header/body), never config fallback.
func resolveInteractiveMCPToken(r *http.Request, bodyToken string) string {
	return userProvidedMCPToken(r, bodyToken)
}
