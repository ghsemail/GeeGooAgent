package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestResolvedWeKnoraDefaults(t *testing.T) {
	t.Setenv("GEEGOO_WEKNORA_API_URL", "")
	t.Setenv("GEEGOO_WEKNORA_WEB_URL", "")
	t.Setenv("GEEGOO_WEKNORA_KB_ID", "")
	t.Setenv("GEEGOO_WEKNORA_API_KEY", "")
	t.Setenv("GEEGOO_WEKNORA_API_KEY_FILE", "")
	cfg := &config.AppConfig{}
	got := cfg.ResolvedWeKnora()
	if got.APIURL != config.DefaultWeKnoraAPIURL {
		t.Fatalf("api_url=%q", got.APIURL)
	}
	if got.WebURL != config.DefaultWeKnoraWebURL {
		t.Fatalf("web_url=%q", got.WebURL)
	}
	if got.KBID != config.DefaultWeKnoraKBID {
		t.Fatalf("kb_id=%q", got.KBID)
	}
	if got.APIKeyFile != config.DefaultWeKnoraAPIKeyFile {
		t.Fatalf("key_file=%q", got.APIKeyFile)
	}
}

func TestResolvedWeKnoraEnvAndKeyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "bff.key")
	if err := os.WriteFile(keyPath, []byte(" sk-from-file \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GEEGOO_WEKNORA_API_URL", "http://wk.local:3481/")
	t.Setenv("GEEGOO_WEKNORA_API_KEY_FILE", keyPath)
	cfg := &config.AppConfig{
		WeKnora: config.WeKnoraConfig{
			KBID: "kb-config",
		},
	}
	got := cfg.ResolvedWeKnora()
	if got.APIURL != "http://wk.local:3481" {
		t.Fatalf("api_url=%q", got.APIURL)
	}
	if got.KBID != "kb-config" {
		t.Fatalf("kb_id=%q", got.KBID)
	}
	if got.APIKey != "sk-from-file" {
		t.Fatalf("api_key=%q", got.APIKey)
	}
}
