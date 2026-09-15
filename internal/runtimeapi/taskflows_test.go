package runtimeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/taskflow"
)

func TestListTaskflowsEndpoint(t *testing.T) {
	h := NewHandler(&app.App{}, "")
	mux := http.NewServeMux()
	h.registerTaskflowRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/taskflows", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	raw, ok := payload["taskflows"].([]any)
	if !ok || len(raw) == 0 {
		t.Fatalf("taskflows=%v", payload["taskflows"])
	}
}

func TestTaskflowsDashboardPayload(t *testing.T) {
	items := taskflowsDashboardPayload()
	if len(items) == 0 {
		t.Fatal("empty taskflows payload")
	}
	if items[0]["id"] != taskflow.TemplateMultiStrategyCompare {
		t.Fatalf("first=%v", items[0]["id"])
	}
}
