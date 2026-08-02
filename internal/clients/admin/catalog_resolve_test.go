package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryModelFromTargetsResolvesByName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/getModel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"model_id":     "6a6db4b74a904218075c19cf",
				"name":         "MiniMax-M3",
				"display_name": "MiniMax M3",
				"type":         "configured",
			},
		})
	})
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["model_id"] != "6a6db4b74a904218075c19cf" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id": "6a6db4b74a904218075c19cf",
			"name":     "MiniMax-M3",
			"token":    "sk-m3",
			"base_url": "https://api.minimax.io/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	targets := []QueryTarget{{BaseURL: srv.URL}}
	doc, _, err := QueryModelFromTargets(context.Background(), targets, "minimax-m3", false)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "MiniMax-M3" || doc.Token != "sk-m3" {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}
