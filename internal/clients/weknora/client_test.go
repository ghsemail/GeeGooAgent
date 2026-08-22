package weknora

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListSearchAndFolders(t *testing.T) {
	t.Parallel()
	var sawKey, sawSearch bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "sk-test" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		sawKey = true
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.URL.Path == "/api/v1/knowledge-bases/kb-1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"kb-1","name":"GeeGoo","embedding_model_id":"emb-1","summary_model_id":"chat-1"}}`))
		case r.URL.Path == "/api/v1/models/emb-1":
			_, _ = w.Write([]byte(`{"data":{"id":"emb-1","name":"kinfra-text-embedding-4b","type":"embedding"}}`))
		case r.URL.Path == "/api/v1/knowledge-bases/kb-1/knowledge/folders":
			_, _ = w.Write([]byte(`{"success":true,"data":{"root_document_count":0,"total_document_count":1,"folders":[{"path":"策略","name":"策略","document_count":1,"total_count":1,"children":[]}]}}`))
		case r.URL.Path == "/api/v1/knowledge-bases/kb-1/knowledge":
			_, _ = w.Write([]byte(`{"success":true,"total":1,"data":[{"id":"doc-1","file_name":"4 Hour MACD Forex Strategy.pdf","folder_path":"策略","file_size":650428,"parse_status":"completed","updated_at":"2026-08-21T00:00:00Z"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/knowledge-search":
			sawSearch = true
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if payload["query"] != "MACD" {
				t.Errorf("query=%v", payload["query"])
			}
			_, _ = w.Write([]byte(`{"success":true,"data":[{"content":"Use 4 hour MACD","knowledge_filename":"4 Hour MACD Forex Strategy.pdf","knowledge_title":"MACD","folder_path":"策略"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "kb-1", srv.Client())
	ctx := context.Background()
	if err := c.Health(ctx); err != nil {
		t.Fatal(err)
	}
	kb, err := c.GetKnowledgeBase(ctx)
	if err != nil || kb.Name != "GeeGoo" {
		t.Fatalf("kb=%+v err=%v", kb, err)
	}
	model, err := c.GetModel(ctx, "emb-1")
	if err != nil || model.Name != "kinfra-text-embedding-4b" {
		t.Fatalf("model=%+v err=%v", model, err)
	}
	tree, err := c.Folders(ctx)
	if err != nil || len(tree.Folders) != 1 || tree.Folders[0].Path != "策略" {
		t.Fatalf("tree=%+v err=%v", tree, err)
	}
	docs, err := c.ListDocuments(ctx, ListDocumentsOpts{})
	if err != nil || len(docs) != 1 || docs[0].ParseStatus != "completed" {
		t.Fatalf("docs=%+v err=%v", docs, err)
	}
	hits, err := c.Search(ctx, "MACD", "", 5)
	if err != nil || len(hits) != 1 || hits[0].Filename == "" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if !sawKey || !sawSearch {
		t.Fatal("expected API key and search request")
	}
}
