package tools

// Entitlements filters tool and skill access per trading user (future: mcp_token → user_id → ACL).
// Today chat allowlists use toolset ChatDefault + chatExcludedTools; wire Entitlements here when ready.
type Entitlements interface {
	AllowedTools(userID string, candidates []string) []string
}

// NoopEntitlements passes through all candidates (current behavior).
type NoopEntitlements struct{}

func (NoopEntitlements) AllowedTools(_ string, candidates []string) []string {
	return candidates
}
