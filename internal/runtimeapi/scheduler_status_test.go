package runtimeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func TestSchedulerStatusReturnsJobs(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	outDir := filepath.ToSlash(dir)
	if err := os.WriteFile(cfgPath, []byte(`{"output_dir":"`+outDir+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	jobsDir := filepath.Join(dir, "scheduler")
	if err := scheduler.SaveJobs(jobsDir, scheduler.DefaultJobs()); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(&app.App{}, cfgPath)
	mux := http.NewServeMux()
	h.registerSchedulerStatusRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/scheduler/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("payload=%v", payload)
	}
	jobs, _ := payload["jobs"].([]any)
	if len(jobs) == 0 {
		t.Fatal("expected default jobs")
	}
}
