package config

// ProfileConfig holds per-profile overrides (Hermes profile isolation parity).
type ProfileConfig struct {
	OutputDir    string   `json:"output_dir,omitempty"`
	MCPToken     string   `json:"mcp_token,omitempty"`
	ChatToolsets []string `json:"chat_toolsets,omitempty"`
	DryRun       *bool    `json:"dry_run,omitempty"`
}
