package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryConfigured(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id":     "abc",
			"name":         "gpt-5.5",
			"display_name": "gpt-5.5",
			"type":         "configured",
			"token":        "sk-test",
			"base_url":     "https://skimtoken.com/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	doc, err := QueryConfigured(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "gpt-5.5" || doc.Token != "sk-test" || doc.BaseURL == "" {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}
