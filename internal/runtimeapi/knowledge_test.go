package runtimeapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestKnowledgeOverviewWithMockWeKnora(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/v1/knowledge-bases/kb-test":
			_, _ = w.Write([]byte(`{"data":{"id":"kb-test","name":"GeeGoo","embedding_model_id":"emb-1","summary_model_id":"chat-1"}}`))
		case "/api/v1/models/emb-1":
			_, _ = w.Write([]byte(`{"data":{"name":"kinfra-text-embedding-4b"}}`))
		case "/api/v1/models/chat-1":
			_, _ = w.Write([]byte(`{"data":{"name":"MiniMax-M3"}}`))
		case "/api/v1/knowledge-bases/kb-test/knowledge":
			_, _ = w.Write([]byte(`{"total":1,"data":[{"id":"d1","file_name":"4 Hour MACD Forex Strategy.pdf","folder_path":"策略","file_size":650428,"parse_status":"completed","updated_at":"2026-08-21T00:00:00Z"}]}`))
		case "/api/v1/knowledge-bases/kb-test/knowledge/folders":
			_, _ = w.Write([]byte(`{"data":{"root_document_count":0,"total_document_count":1,"folders":[{"path":"策略","name":"策略","document_count":1,"total_count":1,"children":[]}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mock.Close()

	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		WeKnora: config.WeKnoraConfig{
			APIURL: mock.URL,
			WebURL: "http://example.test:3480",
			KBID:   "kb-test",
			APIKey: "sk-test",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/knowledge/overview", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true {
		t.Fatalf("body=%v", body)
	}
	if body["kb_name"] != "GeeGoo" {
		t.Fatalf("kb_name=%v", body["kb_name"])
	}
	if int(body["document_count"].(float64)) != 1 {
		t.Fatalf("document_count=%v", body["document_count"])
	}
	if body["chat_model"] != "MiniMax-M3" {
		t.Fatalf("chat_model=%v", body["chat_model"])
	}
	if body["web_url"] != "http://example.test:3480" {
		t.Fatalf("web_url=%v", body["web_url"])
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/knowledge/tree", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tree status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	tree, _ := body["tree"].(map[string]any)
	folders, _ := tree["folders"].([]any)
	if len(folders) != 1 {
		t.Fatalf("folders=%v", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/knowledge/documents?folder_path=%E7%AD%96%E7%95%A5", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("docs status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestKnowledgeOverviewUnreachable(t *testing.T) {
	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		WeKnora: config.WeKnoraConfig{
			APIURL: "http://127.0.0.1:1",
			WebURL: "http://example.test:3480",
			KBID:   "kb-test",
			APIKey: "sk-test",
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/knowledge/overview", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != false {
		t.Fatalf("expected ok=false, got %v", body)
	}
	if body["web_url"] != "http://example.test:3480" {
		t.Fatalf("web_url=%v", body["web_url"])
	}
}
