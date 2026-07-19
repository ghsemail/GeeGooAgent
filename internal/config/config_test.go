package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultPathPrefersEnv(t *testing.T) {
	t.Setenv("GEEGOO_CONFIG", "/tmp/custom.json")
	if got := DefaultPath(); got != "/tmp/custom.json" {
		t.Fatalf("DefaultPath() = %q, want /tmp/custom.json", got)
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-test",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-test",
		"mcp_token": "user-token",
		"output_dir": "` + filepath.ToSlash(dir) + `/data"
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MCPURL() != "http://127.0.0.1:3120" {
		t.Fatalf("MCPURL = %q", cfg.MCPURL())
	}
	if cfg.MCPAPIKey() != "sk-test" {
		t.Fatalf("MCPAPIKey = %q", cfg.MCPAPIKey())
	}
	if cfg.MCPToken() != "user-token" {
		t.Fatalf("MCPToken = %q", cfg.MCPToken())
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://old:3120",
		"api_key": "sk-old",
		"geegoo_url": "http://old:3120",
		"geegoo_api_key": "sk-old",
		"mcp_token": "old-token"
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GEEGOO_BOT_MCP_URL", "http://new:3120")
	t.Setenv("GEEGOO_BOT_MCP_API_KEY", "sk-new")
	t.Setenv("MCP_TOKEN", "new-token")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MCPURL() != "http://new:3120" {
		t.Fatalf("MCPURL = %q", cfg.MCPURL())
	}
	if cfg.MCPAPIKey() != "sk-new" {
		t.Fatalf("MCPAPIKey = %q", cfg.MCPAPIKey())
	}
	if cfg.MCPToken() != "new-token" {
		t.Fatalf("MCPToken = %q", cfg.MCPToken())
	}
}

func asConfigError(err error, target **ConfigError) bool {
	if e, ok := err.(*ConfigError); ok {
		*target = e
		return true
	}
	return false
}

func TestEffectiveCompressionDefaults(t *testing.T) {
	cfg := &AppConfig{}
	c := cfg.EffectiveCompression()
	if !c.Enabled {
		t.Fatal("enabled default true")
	}
	if c.Threshold != 0.5 || c.HygieneThreshold != 0.85 || c.TargetRatio != 0.2 || c.ProtectLastN != 20 {
		t.Fatalf("defaults: %+v", c)
	}
	if c.ProtectFirstN != 3 || c.ContextLength != 128000 || c.ClearToolMinChars != 200 {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestLoadCompressionJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk",
		"compression": {"enabled": false, "threshold": 0.6, "context_length": 64000},
		"auxiliary": {"compression": {"provider": "deepseek", "model": "deepseek-chat"}}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Compression.Enabled == nil || *cfg.Compression.Enabled {
		t.Fatal("want enabled=false")
	}
	c := cfg.EffectiveCompression()
	if c.Enabled || c.Threshold != 0.6 || c.ContextLength != 64000 {
		t.Fatalf("got %+v", c)
	}
	aux := cfg.EffectiveAuxiliaryCompression()
	if aux.Provider != "deepseek" || aux.Model != "deepseek-chat" {
		t.Fatalf("aux %+v", aux)
	}
}

func TestEffectiveMaxTokensThinkingFloor(t *testing.T) {
	cfg := &LLMConfig{MaxTokens: 4096}
	if got := cfg.EffectiveMaxTokens(true); got != 8192 {
		t.Fatalf("thinking floor: got %d", got)
	}
	if got := cfg.EffectiveMaxTokens(false); got != 4096 {
		t.Fatalf("non-thinking: got %d", got)
	}
	cfg.MaxTokens = 16000
	if got := cfg.EffectiveMaxTokens(true); got != 16000 {
		t.Fatalf("respect higher: got %d", got)
	}
}

func TestEffectiveMaxSteps(t *testing.T) {
	if got := (&AppConfig{}).EffectiveMaxSteps(); got != 80 {
		t.Fatalf("default EffectiveMaxSteps = %d, want 80", got)
	}
	if got := (&AppConfig{MaxSteps: 12}).EffectiveMaxSteps(); got != 12 {
		t.Fatalf("EffectiveMaxSteps(12) = %d, want 12", got)
	}
	if got := (&AppConfig{MaxSteps: 200}).EffectiveMaxSteps(); got != 90 {
		t.Fatalf("EffectiveMaxSteps capped = %d, want 90", got)
	}
}

func TestEffectiveSubAgentMaxSteps(t *testing.T) {
	if got := (&AppConfig{}).EffectiveSubAgentMaxSteps(); got != 20 {
		t.Fatalf("default EffectiveSubAgentMaxSteps = %d, want 20", got)
	}
	if got := (&AppConfig{SubAgentMaxSteps: 12}).EffectiveSubAgentMaxSteps(); got != 12 {
		t.Fatalf("EffectiveSubAgentMaxSteps(12) = %d, want 12", got)
	}
	if got := (&AppConfig{SubAgentMaxSteps: 99}).EffectiveSubAgentMaxSteps(); got != 40 {
		t.Fatalf("EffectiveSubAgentMaxSteps capped = %d, want 40", got)
	}
}

func TestEffectiveMCPMaxParallel(t *testing.T) {
	if got := (&AppConfig{}).EffectiveMCPMaxParallel(); got != 6 {
		t.Fatalf("default = %d, want 6", got)
	}
	if got := (&AppConfig{MCPMaxParallel: 10}).EffectiveMCPMaxParallel(); got != 10 {
		t.Fatalf("custom = %d, want 10", got)
	}
}

func TestEffectiveToolMaxParallel(t *testing.T) {
	if got := (&AppConfig{}).EffectiveToolMaxParallel(); got != 4 {
		t.Fatalf("default EffectiveToolMaxParallel = %d, want 4", got)
	}
	if got := (&AppConfig{ToolMaxParallel: 8}).EffectiveToolMaxParallel(); got != 8 {
		t.Fatalf("EffectiveToolMaxParallel(8) = %d, want 8", got)
	}
	if got := (&AppConfig{ToolMaxParallel: 99}).EffectiveToolMaxParallel(); got != 16 {
		t.Fatalf("EffectiveToolMaxParallel capped = %d, want 16", got)
	}
}

func TestEffectiveToolTimeout(t *testing.T) {
	if got := (&AppConfig{}).EffectiveToolTimeout(); got != 120*time.Second {
		t.Fatalf("default EffectiveToolTimeout = %v, want 120s", got)
	}
	if got := (&AppConfig{ToolTimeoutSec: 30}).EffectiveToolTimeout(); got != 30*time.Second {
		t.Fatalf("EffectiveToolTimeout(30) = %v, want 30s", got)
	}
	if got := (&AppConfig{ToolTimeoutSec: 9999}).EffectiveToolTimeout(); got != 600*time.Second {
		t.Fatalf("EffectiveToolTimeout capped = %v, want 600s", got)
	}
}

func TestEffectiveAdvisor(t *testing.T) {
	def := (&AppConfig{}).EffectiveAdvisor()
	if def.Enabled || def.BaseURL != "" || def.Timeout != 3*time.Second || !def.Ranker || !def.Evaluator {
		t.Fatalf("default advisor: %+v", def)
	}
	on := true
	cfg := &AppConfig{Advisor: AdvisorConfig{
		Enabled: &on, BaseURL: "http://127.0.0.1:3410", TimeoutSec: 5,
	}}
	got := cfg.EffectiveAdvisor()
	if !got.Enabled || got.BaseURL != "http://127.0.0.1:3410" || got.Timeout != 5*time.Second {
		t.Fatalf("custom advisor: %+v", got)
	}
}
