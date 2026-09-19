package runtimeapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagramsCatalogAndView(t *testing.T) {
	handler := testCockpitHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/diagrams", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["total"] == nil {
		t.Fatalf("missing total: %v", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/diagrams/workflow.strategy_dev/view?token=test-runtime-key", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("view status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type=%s", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "<svg") {
		t.Fatal("expected svg in view")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/diagrams/workflow.strategy_dev", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/diagrams/workflow.generate_strategy_archive", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("generate_strategy_archive status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/diagrams/workflow.generate_strategy_cognition", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy cognition id status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDashboardWorkflowDetailHasDiagram(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/data", nil)
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
	skills, _ := body["skills"].([]any)
	found := false
	for _, raw := range skills {
		row, _ := raw.(map[string]any)
		if row["name"] != "premarket_market" {
			continue
		}
		detail, _ := row["workflow_detail"].(map[string]any)
		diagram, _ := detail["diagram"].(map[string]any)
		if diagram["id"] == "workflow.premarket_market" {
			found = true
		}
	}
	if !found {
		t.Fatalf("premarket_market diagram missing in skills=%v", skills)
	}
}
