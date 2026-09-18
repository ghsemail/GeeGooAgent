package runtimeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryPlanGet(t *testing.T) {
	h := NewHandler(nil, "")
	mux := http.NewServeMux()
	h.registerMemoryPlanRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/memory/plan", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["style"] != "cursor" {
		t.Fatalf("style=%v", payload["style"])
	}
	sections, ok := payload["sections"].([]any)
	if !ok || len(sections) < 4 {
		t.Fatalf("sections=%v", payload["sections"])
	}
	first, ok := sections[0].(map[string]any)
	if !ok || first["id"] != "overview" {
		t.Fatalf("first section=%v", sections[0])
	}
}
