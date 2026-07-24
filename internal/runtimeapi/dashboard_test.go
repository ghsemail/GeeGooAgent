package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashboardData(t *testing.T) {
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
	if body["provider"] == nil {
		t.Fatalf("missing provider: %v", body)
	}
}

func TestDashboardEvents(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/events", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDashboardQueryRejectsWrite(t *testing.T) {
	handler := testCockpitHandler(t)
	body := bytes.NewBufferString(`{"sql":"DELETE FROM agent_sessions"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/dashboard/query", body)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompareHistory(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/compare/history", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
