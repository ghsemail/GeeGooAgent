package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) registerEvalWorkflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/dashboard/eval/strategy-catalog", h.evalStrategyCatalog)
	mux.HandleFunc("POST /v1/dashboard/eval/cases/{id}/prepare", h.evalCasePrepare)
}

func (h *Handler) evalStrategyCatalog(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.App == nil || h.App.Config == nil {
		writeError(w, http.StatusServiceUnavailable, "agent config not ready")
		return
	}
	types := r.URL.Query()["type"]
	if len(types) == 0 {
		if raw := strings.TrimSpace(r.URL.Query().Get("types")); raw != "" {
			types = strings.Split(raw, ",")
		}
	}
	mcpToken := resolveChatMCPToken(r, "", h.configMCPToken())
	entries, err := eval.ListStrategyCatalog(
		r.Context(),
		h.App.Config.SignalCatalogURL(),
		h.App.Config.SignalCatalogAPIKey(),
		mcpToken,
		types,
	)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]any{"items": entries, "total": len(entries)})
}

type evalCasePrepareRequest struct {
	RandomStrategyEnabled *bool  `json:"random_strategy_enabled"`
	StrategyName          string `json:"strategy_name"`
	StrategyCatalogType   string `json:"strategy_catalog_type"`
}

func (h *Handler) evalCasePrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	caseID := strings.TrimSpace(r.PathValue("id"))
	opts, title, err := h.loadTurnPlanCaseOptions(r, caseID)
	if err != nil {
		if err == errEvalCaseNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req evalCasePrepareRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.RandomStrategyEnabled != nil {
		opts.RandomStrategyEnabled = *req.RandomStrategyEnabled
	}
	if name := strings.TrimSpace(req.StrategyName); name != "" {
		opts.StrategyName = name
	}
	if typ := strings.TrimSpace(req.StrategyCatalogType); typ != "" {
		opts.StrategyCatalogType = typ
	}
	mcpToken := resolveChatMCPToken(r, "", h.configMCPToken())
	opts, picked, err := h.resolveEvalCaseOptions(r.Context(), opts, mcpToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	message := eval.ResolveGenerateCognitionMessage(opts, picked)
	writeJSON(w, map[string]any{
		"ok":                    true,
		"case_id":               caseID,
		"title":                 title,
		"message":               message,
		"random_strategy_enabled": opts.RandomStrategyEnabled,
		"strategy_name":         firstNonEmptyString(opts.StrategyName, picked),
		"strategy_catalog_type": opts.StrategyCatalogType,
	})
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
