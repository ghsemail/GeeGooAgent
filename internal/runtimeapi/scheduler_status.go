package runtimeapi

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func (h *Handler) registerSchedulerStatusRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/scheduler/status", h.schedulerStatus)
}

func (h *Handler) schedulerStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
	writeJSONStatus(w, http.StatusInternalServerError, map[string]string{
			"status": "down",
			"detail": err.Error(),
		})
		return
	}
	workspace, err := cfg.ResolveOutputDir()
	if err != nil {
	writeJSONStatus(w, http.StatusInternalServerError, map[string]string{
			"status": "down",
			"detail": err.Error(),
		})
		return
	}
	jobsDir := filepath.Join(workspace, "scheduler")
	jf, err := scheduler.LoadJobs(jobsDir)
	if err != nil {
	writeJSONStatus(w, http.StatusInternalServerError, map[string]string{
			"status": "down",
			"detail": err.Error(),
		})
		return
	}
	if len(jf.Jobs) == 0 {
		jf = scheduler.DefaultJobs()
	}
	type jobRow struct {
		Name        string `json:"name"`
		Skill       string `json:"skill"`
		Spec        string `json:"spec"`
		Enabled     bool   `json:"enabled"`
		LastRunAt   string `json:"last_run_at,omitempty"`
		LastVerdict string `json:"last_verdict,omitempty"`
	}
	rows := make([]jobRow, 0, len(jf.Jobs))
	for _, j := range jf.Jobs {
		rows = append(rows, jobRow{
			Name:        j.Name,
			Skill:       j.Skill,
			Spec:        j.Cron,
			Enabled:     j.Enabled,
			LastRunAt:   j.LastRun,
			LastVerdict: j.LastVerdict,
		})
	}
	writeJSONStatus(w, http.StatusOK, map[string]any{
		"status": "ok",
		"jobs":   rows,
	})
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
