package runtimeapi

import (
	"net/http"

	"github.com/ghsemail/GeeGooAgent/internal/taskflow"
)

func (h *Handler) registerTaskflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/taskflows", h.listTaskflows)
}

func (h *Handler) listTaskflows(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"taskflows": taskflow.ListTemplatesPayload(),
	})
}

func taskflowsDashboardPayload() []map[string]any {
	return taskflow.ListTemplatesPayload()
}
