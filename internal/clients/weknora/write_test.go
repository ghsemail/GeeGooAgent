package weknora

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateManualKnowledge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/knowledge-bases/kb-1/knowledge" {
			_, _ = w.Write([]byte(`{"success":true,"data":[],"total":0}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/knowledge-bases/kb-1/knowledge/manual" {
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"kid-1","title":"Macd4H · 策略认知","parse_status":"processing"}}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/knowledge/folder" {
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "kb-1", srv.Client())
	doc, err := c.UpsertManualKnowledge(context.Background(), "策略认知", "Macd4H · 策略认知", "# hello")
	if err != nil {
		t.Fatalf("UpsertManualKnowledge: %v", err)
	}
	if doc.ID != "kid-1" {
		t.Fatalf("id=%s", doc.ID)
	}
}
