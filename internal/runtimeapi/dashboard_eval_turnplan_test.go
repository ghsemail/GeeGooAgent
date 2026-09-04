package runtimeapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvalRunTurnPlanBuiltin(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/dashboard/eval/run-turn-plan", nil)
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
		t.Fatalf("expected ok=true: %v", body)
	}
	if body["plan_only"] != true {
		t.Fatalf("expected plan_only: %v", body)
	}
	total, _ := body["total"].(float64)
	if total < 9 {
		t.Fatalf("expected at least 9 turns, got %v", total)
	}
}

func TestEvalCaseRunTurnPlanRouting(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/dashboard/eval/cases/turn_plan_routing/run", nil)
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
	if body["case_id"] != "turn_plan_routing" {
		t.Fatalf("case_id=%v", body["case_id"])
	}
}

func TestEvalCaseRunRejectsNonTurnPlan(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/dashboard/eval/cases/hello_then_stock_analysis/run", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
