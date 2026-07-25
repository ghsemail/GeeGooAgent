package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestMemorySoulGetPut(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GEEGOO_HOME", home)
	handler := testCockpitHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/memory/soul", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got["content"].(string), "GeeGoo") {
		t.Fatalf("unexpected default soul: %v", got["content"])
	}

	body, _ := json.Marshal(map[string]string{"content": "Custom persona line.\n"})
	req = httptest.NewRequest(http.MethodPut, "/v1/memory/soul", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got["content"].(string), "Custom persona") {
		t.Fatalf("save response missing content: %v", got)
	}
	saved := chatprompt.LoadSoulFromHome(home)
	if !strings.Contains(saved, "Custom persona") {
		t.Fatalf("file not written: %q", saved)
	}
	if got["path"] != filepath.Join(home, "SOUL.md") {
		t.Fatalf("unexpected path: %v", got["path"])
	}
}
