package runtimeapi

import (
	"embed"
	"net/http"
)

//go:embed dashboard_eval_auto.html
var evalAutoPageFS embed.FS

func (h *Handler) evalAutoPage(w http.ResponseWriter, r *http.Request) {
	raw, err := evalAutoPageFS.ReadFile("dashboard_eval_auto.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
