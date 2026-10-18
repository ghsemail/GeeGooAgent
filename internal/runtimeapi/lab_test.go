package runtimeapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestLabRoutesDisabledByDefault(t *testing.T) {
	h := NewHandler(&app.App{Config: &config.AppConfig{}}, "")
	mux := http.NewServeMux()
	h.registerLabRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/lab/experiments", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLabListExperimentsWhenEnabled(t *testing.T) {
	on := true
	h := NewHandler(&app.App{Config: &config.AppConfig{LabEnabledField: &on, UserMCPToken: "tok"}}, "")
	mux := http.NewServeMux()
	h.registerLabRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/lab/experiments", nil)
	req.Header.Set("X-MCP-Token", "tok")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	exps, _ := out["experiments"].([]any)
	if len(exps) != 1 {
		t.Fatalf("experiments=%v", out["experiments"])
	}
}

func TestLabPriceAnalysisBadJSON(t *testing.T) {
	on := true
	h := NewHandler(&app.App{Config: &config.AppConfig{LabEnabledField: &on, UserMCPToken: "tok"}}, "")
	mux := http.NewServeMux()
	h.registerLabRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/lab/experiments/price-analysis", bytes.NewReader([]byte("{")))
	req.Header.Set("X-MCP-Token", "tok")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
