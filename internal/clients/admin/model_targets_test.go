package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryConfiguredWithBearer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "gpt-5.5", "token": "sk-test", "base_url": "https://example.com/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	doc, err := QueryConfiguredWithBearer(context.Background(), srv.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Token != "sk-test" {
		t.Fatalf("token=%q", doc.Token)
	}
}

func TestQueryConfiguredFromTargetsSkipsEmptyToken(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "gpt-5.5"})
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "gpt-5.5", "token": "sk-good", "base_url": "https://skimtoken.com/v1",
		})
	}))
	defer good.Close()

	doc, src, err := QueryConfiguredFromTargets(context.Background(),
		QueryTarget{BaseURL: bad.URL},
		QueryTarget{BaseURL: good.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Token != "sk-good" || src != good.URL {
		t.Fatalf("got %+v from %s", doc, src)
	}
}
