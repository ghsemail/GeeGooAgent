package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSkillRejectsPlaceholderWithoutSteps(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-test",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-test",
		"mcp_token": "user-token",
		"output_dir": "` + filepath.ToSlash(dir) + `/data",
		"dry_run": true,
		"llm": {"provider": "deepseek", "token_key": "test-key"},
		"sandbox": {"allowed_hosts": ["127.0.0.1"]}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	application, err := LoadFromConfigPath(cfgPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	_, err = application.RunSkill("nonexistent_skill")
	if err == nil {
		t.Fatal("expected unknown skill to fail")
	}
	if !strings.Contains(err.Error(), "unknown skill") {
		t.Fatalf("unexpected error: %v", err)
	}
}
