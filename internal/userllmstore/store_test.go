package userllmstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestBackendLoadSaveHTTP(t *testing.T) {
	var gotSave bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/getUserLLMSettings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"llm_settings": map[string]any{"catalog_model_id": "m1"},
			})
		case "/setUserLLMSettings":
			gotSave = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := &config.AppConfig{
		BotServiceAPIURL: srv.URL,
		BotServiceAPIKey: "secret",
	}
	b := NewBackend(cfg, dir)
	if !b.Enabled() {
		t.Fatal("expected backend enabled")
	}
	doc, err := b.Load(context.Background(), "user-1")
	if err != nil || doc == nil || doc.CatalogModelID != "m1" {
		t.Fatalf("load: err=%v doc=%+v", err, doc)
	}
	doc.CatalogModelID = "m2"
	if err := b.Save(context.Background(), "user-1", doc); err != nil {
		t.Fatal(err)
	}
	if !gotSave {
		t.Fatal("expected save HTTP call")
	}
	// DB mode must not create local files.
	path := filepath.Join(dir, "user_llm_settings", "user-1.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected local file: %v", err)
	}
}

func TestBackendFileFallbackWhenNoDB(t *testing.T) {
	dir := t.TempDir()
	b := NewBackend(nil, dir)
	if b.Enabled() {
		t.Fatal("expected disabled without backend config")
	}
	doc := &Settings{CatalogModelID: "local-model"}
	if err := b.Save(context.Background(), "u1", doc); err != nil {
		t.Fatal(err)
	}
	loaded, err := b.Load(context.Background(), "u1")
	if err != nil || loaded == nil || loaded.CatalogModelID != "local-model" {
		t.Fatalf("load: err=%v doc=%+v", err, loaded)
	}
}

func TestMergeEffectiveGateway(t *testing.T) {
	base := config.LLMConfig{Provider: "geegoo", Model: "base"}
	doc := &Settings{
		Gateways: map[string]GatewayEntry{
			"web": {CatalogModelID: "web-id"},
		},
	}
	out := MergeEffective(base, doc, "web")
	if out.CatalogModelID != "web-id" {
		t.Fatalf("got %q", out.CatalogModelID)
	}
}
