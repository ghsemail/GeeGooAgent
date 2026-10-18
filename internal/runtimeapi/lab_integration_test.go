package runtimeapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient/reference"
)

func TestLabPriceAnalysisE2EAgainstReferenceLabAPI(t *testing.T) {
	ref := reference.NewServer()
	labAPI := httptest.NewServer(ref.HTTP.Handler)
	defer labAPI.Close()

	on := true
	cfg := &config.AppConfig{
		LabEnabledField: &on,
		UserMCPToken:    "agent-tok",
		LabAPIURLField:  labAPI.URL,
		LabAPIKeyField:  reference.DefaultAPIKey,
	}
	h := NewHandler(&app.App{Config: cfg}, "")
	mux := http.NewServeMux()
	h.registerLabRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"code":             "00700.HK",
		"signal_frequency": "60m",
		"months":           3,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/lab/experiments/price-analysis", bytes.NewReader(body))
	req.Header.Set("X-MCP-Token", "agent-tok")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["experiment"] != "price-analysis" {
		t.Fatalf("experiment=%v", out["experiment"])
	}
	analysis, _ := out["analysis"].(map[string]any)
	rows, _ := analysis["rows"].([]any)
	if len(rows) == 0 {
		t.Fatalf("analysis=%v", analysis)
	}
	if analysis["price_score"] == nil && analysis["score_reason"] == nil {
		t.Fatalf("expected composite analysis fields: %v", analysis)
	}
}
