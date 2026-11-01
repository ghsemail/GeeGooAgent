package runtimeapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func TestWriteAgentProgressSSESignalDiagnoseResultReady(t *testing.T) {
	payload := map[string]any{
		"session_id": "sess-1",
		"result_key": "sd-res-test",
		"run_id":     "flow-1",
		"phase":      "final",
		"stock_code": "00700.HK",
	}
	w := httptest.NewRecorder()
	var rw http.ResponseWriter = w
	flusher, ok := rw.(http.Flusher)
	if !ok {
		t.Fatal("recorder must implement http.Flusher")
	}
	writeAgentProgressSSE(rw, flusher, workflowchat.EventSignalDiagnoseResultReady, payload)

	body := w.Body.String()
	if !strings.Contains(body, "event: signal_diagnose_result_ready") {
		t.Fatalf("missing SSE event line: %s", body)
	}
	wrapped := runtime.ProgressPayload(workflowchat.EventSignalDiagnoseResultReady, payload)
	if wrapped["item_type"] != "signal_diagnose_result_ready" {
		t.Fatalf("item_type=%v", wrapped["item_type"])
	}
	if strings.Contains(body, "chart_probe") {
		t.Fatalf("must not leak chart_probe: %s", body)
	}
}
