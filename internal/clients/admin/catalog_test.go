package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/getModel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"model_id": "a", "name": "gpt-5.5", "type": "configured"},
			{"model_id": "b", "name": "deepseek-v4", "type": "active"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	docs, err := ListModels(context.Background(), srv.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Name != "gpt-5.5" {
		t.Fatalf("unexpected docs: %+v", docs)
	}
}

func TestQueryModelByID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["model_id"] != "b" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id": "b", "name": "deepseek-v4", "token": "sk-x", "base_url": "https://api.example/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	doc, err := QueryModelByID(context.Background(), srv.URL, "", "b")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "deepseek-v4" || doc.Token != "sk-x" {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}
