package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertEnvFileAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := UpsertEnvFile(path, map[string]string{
		"FEISHU_APP_ID":     "cli_a",
		"FEISHU_APP_SECRET": "sec_a",
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "FEISHU_APP_ID=cli_a") || !strings.Contains(s, "FEISHU_APP_SECRET=sec_a") {
		t.Fatalf("%s", raw)
	}
	_ = os.Unsetenv("FEISHU_APP_ID")
	_ = os.Unsetenv("FEISHU_APP_SECRET")
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FEISHU_APP_ID") != "cli_a" {
		t.Fatalf("got %q", os.Getenv("FEISHU_APP_ID"))
	}
	t.Setenv("FEISHU_APP_ID", "keep")
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FEISHU_APP_ID") != "keep" {
		t.Fatal("should not override")
	}
}
