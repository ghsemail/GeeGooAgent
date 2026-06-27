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
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	Thinking        *bool   `json:"thinking"`
	ReasoningEffort string  `json:"reasoning_effort"`
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

// AppConfig is compatible with Python config.json.
type AppConfig struct {
	BaseURL          string        `json:"base_url"`
	APIKey           string        `json:"api_key"`
	GeeGooURL        string        `json:"geegoo_url"`
	GeeGooAPIKey     string        `json:"geegoo_api_key"`
	UserMCPToken     string        `json:"mcp_token"`
	SignalBaseURL    string        `json:"signal_base_url"`
	DataBaseURL      string        `json:"data_base_url"`
	OutputDir        string        `json:"output_dir"`
	DryRun           bool          `json:"dry_run"`
	FeishuWebhookURL *string       `json:"feishu_webhook_url"`
	MaxSteps         int           `json:"max_steps"`
	LLM              LLMConfig     `json:"llm"`
	Search           SearchConfig  `json:"search"`
	Sandbox          SandboxConfig `json:"sandbox"`
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
