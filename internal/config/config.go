package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LLMConfig mirrors Python llm section.
type LLMConfig struct {
	Provider        string  `json:"provider"`
	TokenKey        string  `json:"token_key"`
	Model           string  `json:"model"`
	BaseURL         string  `json:"base_url,omitempty"` // optional local override; ops configured wins when use_ops_model
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	Thinking        *bool   `json:"thinking"`
	ReasoningEffort string  `json:"reasoning_effort"`
	// UseOpsModel when true/nil: pull configured model from Signal catalog/admin.
	// Set false to force local provider/token_key/model/base_url only.
	UseOpsModel *bool `json:"use_ops_model,omitempty"`
}

// OpsModelEnabled reports whether RebuildGateway should query ops configured model.
func (c *LLMConfig) OpsModelEnabled() bool {
	if c.UseOpsModel == nil {
		return true
	}
	return *c.UseOpsModel
}

// EffectiveMaxTokens returns the chat completion max_tokens.
// Thinking mode needs headroom for reasoning_content; values below 8192 are raised.
func (c *LLMConfig) EffectiveMaxTokens(thinkingEnabled bool) int {
	max := 4096
	if c != nil && c.MaxTokens > 0 {
		max = c.MaxTokens
	}
	if thinkingEnabled && max < 8192 {
		return 8192
	}
	return max
}

// SearchConfig controls free web search (DuckDuckGo by default).
type SearchConfig struct {
	Provider   string `json:"provider"`
	MaxResults int    `json:"max_results"`
}

// SandboxConfig holds HTTP host allowlist.
type SandboxConfig struct {
	AllowedHosts []string `json:"allowed_hosts"`
}

// CompressionConfig is the JSON shape (pointers allow distinguishing unset).
type CompressionConfig struct {
	Enabled           *bool   `json:"enabled,omitempty"`
	Threshold         float64 `json:"threshold,omitempty"`
	TargetRatio       float64 `json:"target_ratio,omitempty"`
	ProtectLastN      int     `json:"protect_last_n,omitempty"`
	ProtectFirstN     int     `json:"protect_first_n,omitempty"`
	ContextLength     int     `json:"context_length,omitempty"`
	ClearToolMinChars int     `json:"clear_tool_min_chars,omitempty"`
}

// AuxiliaryLLMConfig is optional summarizer credentials.
type AuxiliaryLLMConfig struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	TokenKey string `json:"token_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type AuxiliaryConfig struct {
	Compression AuxiliaryLLMConfig `json:"compression"`
}

// ResolvedCompression is EffectiveCompression output (no pointers).
type ResolvedCompression struct {
	Enabled           bool
	Threshold         float64
	TargetRatio       float64
	ProtectLastN      int
	ProtectFirstN     int
	ContextLength     int
	ClearToolMinChars int
}

// AppConfig is compatible with Python config.json.
type AppConfig struct {
	BaseURL          string            `json:"base_url"`
	APIKey           string            `json:"api_key"`
	GeeGooURL        string            `json:"geegoo_url"`
	GeeGooAPIKey     string            `json:"geegoo_api_key"`
	UserMCPToken     string            `json:"mcp_token"`
	SignalBaseURL    string            `json:"signal_base_url"`
	DataBaseURL      string            `json:"data_base_url"`
	OutputDir        string            `json:"output_dir"`
	DryRun           bool              `json:"dry_run"`
	FeishuWebhookURL *string           `json:"feishu_webhook_url"`
	MaxSteps         int               `json:"max_steps"`
	LLM              LLMConfig         `json:"llm"`
	Search           SearchConfig      `json:"search"`
	Sandbox          SandboxConfig     `json:"sandbox"`
	Compression      CompressionConfig `json:"compression"`
	Auxiliary        AuxiliaryConfig   `json:"auxiliary"`
}

// ConfigError indicates invalid or missing configuration.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string { return e.Message }

// Load reads and validates config from path.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigError{Message: fmt.Sprintf("config file not found: %s", path)}
		}
		return nil, &ConfigError{Message: fmt.Sprintf("read config: %v", err)}
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, &ConfigError{Message: fmt.Sprintf("invalid JSON in config: %s", path)}
	}
	if cfg.OutputDir != "" {
		_ = os.MkdirAll(cfg.OutputDir, 0o755)
	}
	applyEnv(&cfg)
	return &cfg, nil
}

func applyEnv(cfg *AppConfig) {
	if v := os.Getenv("GEEGOO_BOT_MCP_URL"); v != "" {
		cfg.GeeGooURL = v
		cfg.BaseURL = v
	}
	if v := os.Getenv("GEEGOO_BOT_MCP_API_KEY"); v != "" {
		cfg.GeeGooAPIKey = v
		cfg.APIKey = v
	}
	if v := os.Getenv("MCP_TOKEN"); v != "" {
		cfg.UserMCPToken = v
	}
	if v := os.Getenv("GEEGOO_WEB_SEARCH"); v != "" {
		cfg.Search.Provider = v
	}
}

func (c *AppConfig) EffectiveCompression() ResolvedCompression {
	out := ResolvedCompression{
		Enabled: true, Threshold: 0.5, TargetRatio: 0.2,
		ProtectLastN: 20, ProtectFirstN: 3, ContextLength: 128000, ClearToolMinChars: 200,
	}
	if c == nil {
		return out
	}
	src := c.Compression
	if src.Enabled != nil {
		out.Enabled = *src.Enabled
	}
	if src.Threshold > 0 {
		out.Threshold = src.Threshold
	}
	if src.TargetRatio > 0 {
		out.TargetRatio = src.TargetRatio
	}
	if src.ProtectLastN > 0 {
		out.ProtectLastN = src.ProtectLastN
	}
	if src.ProtectFirstN > 0 {
		out.ProtectFirstN = src.ProtectFirstN
	}
	if src.ContextLength > 0 {
		out.ContextLength = src.ContextLength
	}
	if src.ClearToolMinChars > 0 {
		out.ClearToolMinChars = src.ClearToolMinChars
	}
	if out.Threshold > 1 {
		out.Threshold = 1
	}
	if out.TargetRatio < 0.1 {
		out.TargetRatio = 0.1
	}
	if out.TargetRatio > 0.8 {
		out.TargetRatio = 0.8
	}
	return out
}

// EffectiveAuxiliaryCompression returns aux fields with empty → main LLM fallback.
func (c *AppConfig) EffectiveAuxiliaryCompression() AuxiliaryLLMConfig {
	var aux AuxiliaryLLMConfig
	if c == nil {
		return aux
	}
	aux = c.Auxiliary.Compression
	if strings.TrimSpace(aux.Provider) == "" {
		aux.Provider = c.LLM.Provider
	}
	if strings.TrimSpace(aux.Model) == "" {
		aux.Model = c.LLM.Model
	}
	if strings.TrimSpace(aux.TokenKey) == "" {
		aux.TokenKey = c.LLM.TokenKey
	}
	if strings.TrimSpace(aux.BaseURL) == "" {
		aux.BaseURL = c.LLM.BaseURL
	}
	return aux
}

// EffectiveSearch returns search settings with defaults (duckduckgo, max 5).
func (c *AppConfig) EffectiveSearch() SearchConfig {
	out := c.Search
	if strings.TrimSpace(out.Provider) == "" {
		out.Provider = "duckduckgo"
	}
	if out.MaxResults <= 0 {
		out.MaxResults = 5
	}
	return out
}

// MCPURL returns the MCP base URL (env override > geegoo_url > base_url).
func (c *AppConfig) MCPURL() string {
	if v := os.Getenv("GEEGOO_BOT_MCP_URL"); v != "" {
		return trimSlash(v)
	}
	if c.GeeGooURL != "" {
		return trimSlash(c.GeeGooURL)
	}
	return trimSlash(c.BaseURL)
}

// MCPAPIKey returns Bearer key for MCP.
func (c *AppConfig) MCPAPIKey() string {
	if v := os.Getenv("GEEGOO_BOT_MCP_API_KEY"); v != "" {
		return v
	}
	if c.GeeGooAPIKey != "" {
		return c.GeeGooAPIKey
	}
	return c.APIKey
}

// MCPToken returns user identity token for request body.
func (c *AppConfig) MCPToken() string {
	if v := os.Getenv("MCP_TOKEN"); v != "" {
		return v
	}
	return c.UserMCPToken
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// ResolveOutputDir returns absolute output directory.
func (c *AppConfig) ResolveOutputDir() (string, error) {
	if c.OutputDir == "" {
		return filepath.Abs(filepath.Join(Home(), "data"))
	}
	return filepath.Abs(c.OutputDir)
}
