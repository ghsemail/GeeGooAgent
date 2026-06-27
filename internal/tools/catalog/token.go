package catalog

// NeedsMCPToken reports whether a catalog HTTP tool must send mcp_token.
// Matches Python HttpToolSpec default (true) with explicit opt-out for public endpoints.
func NeedsMCPToken(name string) bool {
	switch name {
	case "search_code", "get_index_signals", "get_signal_combinations":
		return false
	default:
		return true
	}
}
