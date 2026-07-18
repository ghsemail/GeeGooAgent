package doctor_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/doctor"
)

func captureDoctor(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestDoctorShowsProfileLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-real-key",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-real-key",
		"mcp_token": "user-mcp",
		"active_profile": "work",
		"profiles": {
			"work": { "output_dir": "/tmp/work", "chat_toolsets": ["market"] }
		},
		"llm": {"token_key": "llm-key"}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out := captureDoctor(func() {
		code := doctor.RunWithOptions(path, doctor.Options{SkipConnectivity: true})
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
	})
	if !containsAll(out, "[OK] profile:", "work via active_profile", "output_dir=/tmp/work") {
		t.Fatalf("output missing profile line:\n%s", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !bytes.Contains([]byte(s), []byte(p)) {
			return false
		}
	}
	return true
}
