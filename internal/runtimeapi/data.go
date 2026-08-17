package runtimeapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (h *Handler) registerDataRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/data/overview", h.dataOverview)
	mux.HandleFunc("GET /v1/data/nodes", h.dataNodes)
	mux.HandleFunc("GET /v1/data/nodes/{id}/news/sources", h.dataNodeNewsSources)
	mux.HandleFunc("PATCH /v1/data/nodes/{id}/news/sources", h.dataNodeNewsSourcesPatch)
	mux.HandleFunc("GET /v1/data/nodes/{id}/news/health", h.dataNodeNewsHealth)
	mux.HandleFunc("GET /v1/data/nodes/{id}/market/capabilities", h.dataNodeMarketCapabilities)
	mux.HandleFunc("POST /v1/data/probe", h.dataProbe)
}

func (h *Handler) dataOverview(w http.ResponseWriter, r *http.Request) {
	force := strings.EqualFold(r.URL.Query().Get("force"), "true")
	ctx, cancel := context.WithTimeout(r.Context(), dataBFFOverviewBudget)
	defer cancel()
	payload, err := h.collectDataOverview(ctx, force)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, payload)
}

func (h *Handler) dataNodes(w http.ResponseWriter, r *http.Request) {
	cfg := h.appConfig()
	nodes := cfg.ResolvedDataNodes()
	probe := strings.EqualFold(r.URL.Query().Get("probe"), "true")

	out := make([]map[string]any, 0, len(nodes))
	collector := newDataFleetCollector()
	ctx, cancel := context.WithTimeout(r.Context(), dataBFFOverviewBudget)
	defer cancel()

	for _, node := range nodes {
		item := map[string]any{
			"id":       node.ID,
			"label":    node.Label,
			"base_url": node.BaseURL,
			"regions":  node.Regions,
		}
		if probe {
			probed := collector.probeNode(ctx, node)
			raw, _ := json.Marshal(probed)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			item["health"] = m["health"]
			item["news"] = m["news"]
			item["capabilities"] = m["capabilities"]
		}
		out = append(out, item)
	}
	writeJSON(w, map[string]any{"nodes": out})
}

func (h *Handler) dataNodeNewsSources(w http.ResponseWriter, r *http.Request) {
	h.proxyDataNode(w, r, http.MethodGet, "/v1/news/sources", nil)
}

func (h *Handler) dataNodeNewsSourcesPatch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.proxyDataNode(w, r, http.MethodPatch, "/v1/news/sources", body)
	globalDataFleetCache.clear()
}

func (h *Handler) dataNodeNewsHealth(w http.ResponseWriter, r *http.Request) {
	h.proxyDataNode(w, r, http.MethodGet, "/v1/news/health", nil)
}

func (h *Handler) dataNodeMarketCapabilities(w http.ResponseWriter, r *http.Request) {
	h.proxyDataNode(w, r, http.MethodGet, "/v1/market/capabilities", nil)
}

func (h *Handler) proxyDataNode(w http.ResponseWriter, r *http.Request, method, path string, body []byte) {
	nodeID := strings.TrimSpace(r.PathValue("id"))
	node, ok := h.appConfig().DataNodeByID(nodeID)
	if !ok {
		writeError(w, http.StatusNotFound, "data node not found: "+nodeID)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dataBFFNodeTimeout)
	defer cancel()
	payload, status, err := newDataFleetCollector().proxy(ctx, node, method, path, body)
	if err != nil {
		if status >= 400 && status < 600 {
			writeError(w, status, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, payload)
}
