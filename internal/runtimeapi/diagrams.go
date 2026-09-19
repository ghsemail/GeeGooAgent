package runtimeapi

import (
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/diagram"
)

func (h *Handler) registerDiagramRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/diagrams", h.diagramsList)
	mux.HandleFunc("GET /v1/diagrams/{id}/view", h.diagramView)
	mux.HandleFunc("GET /v1/diagrams/{id}", h.diagramGet)
}

func (h *Handler) diagramService() *diagram.Default {
	root := "."
	if h != nil && h.App != nil {
		if r := strings.TrimSpace(h.App.ProjectRoot()); r != "" {
			root = r
		}
	}
	return diagram.New(root)
}

func (h *Handler) diagramsList(w http.ResponseWriter, r *http.Request) {
	svc := h.diagramService()
	items := svc.List()
	writeJSON(w, map[string]any{"diagrams": items, "total": len(items)})
}

func (h *Handler) diagramGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	doc, err := h.diagramService().Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, map[string]any{
		"id":    doc.ID,
		"title": doc.Title,
		"type":  doc.Kind,
		"ir":    doc.IR,
	})
}

func (h *Handler) diagramView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	art, err := h.diagramService().Render(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "unknown id") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusServiceUnavailable, err.Error()+"; run go generate ./internal/diagram")
		return
	}
	w.Header().Set("Content-Type", art.ContentType)
	if art.ContentType == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'self' http: https:")
	w.Header().Set("X-Diagram-Source", art.Source)
	w.Header().Set("X-Diagram-SHA256", art.SHA256)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(art.HTML)
}
