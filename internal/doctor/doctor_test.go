package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/doctor"
)

func TestDoctorMissingConfig(t *testing.T) {
	code := doctor.Run(filepath.Join(t.TempDir(), "missing.json"))
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestDoctorValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-real-key",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-real-key",
		"mcp_token": "user-mcp",
		"llm": {"token_key": "llm-key"}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	code := doctor.Run(path)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}
