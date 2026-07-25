package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type dashboardSettingsRequest struct {
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	CatalogModelID string   `json:"catalog_model_id"`
	UseOpsModel    *bool    `json:"use_ops_model"`
	Thinking       string   `json:"thinking"` // on | off | auto
	Temperature    *float64 `json:"temperature"`
	MaxTokens      *int     `json:"max_tokens"`
	Pinned         []string `json:"pinned"`
}

func (h *Handler) registerSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/dashboard/settings", h.dashboardApplySettings)
	mux.HandleFunc("GET /v1/dashboard/models", h.dashboardListModels)
}

func (h *Handler) dashboardListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.fetchCatalogModels(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(models))
	for _, m := range models {
		label := llm.CatalogModelLabel(m)
		out = append(out, map[string]any{
			"model_id":     m.ModelID,
			"name":         m.Name,
			"display_name": m.DisplayName,
			"label":        label,
			"type":         m.Type,
			"provider":     m.Provider,
			"configured":   m.Type == "configured",
		})
	}
	writeJSON(w, map[string]any{"models": out})
}

func (h *Handler) dashboardApplySettings(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Config == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return
	}
	var req dashboardSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	userID := resolveUserID(r)
	if userID != "" {
		us, err := loadUserLLMSettings(h.userSettingsOutputDir(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if us == nil {
			us = &userLLMSettings{}
		}
		us.applyRequest(req)
		if err := saveUserLLMSettings(h.userSettingsOutputDir(), userID, us); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		info, err := h.buildSettingsInfo(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "settings": info})
		return
	}

	llmCfg := h.App.Config.LLM
	if p := strings.TrimSpace(req.Provider); p != "" {
		llmCfg.Provider = p
	}
	if m := strings.TrimSpace(req.Model); m != "" {
		llmCfg.Model = m
	}
	if id := strings.TrimSpace(req.CatalogModelID); id != "" {
		llmCfg.CatalogModelID = id
		useOps := true
		llmCfg.UseOpsModel = &useOps
	} else if req.UseOpsModel != nil {
		llmCfg.UseOpsModel = req.UseOpsModel
		if *req.UseOpsModel {
			llmCfg.CatalogModelID = ""
		}
	}
	switch strings.ToLower(strings.TrimSpace(req.Thinking)) {
	case "on":
		v := true
		llmCfg.Thinking = &v
	case "off":
		v := false
		llmCfg.Thinking = &v
	case "auto", "":
		llmCfg.Thinking = nil
	}
	if req.Temperature != nil && *req.Temperature > 0 {
		llmCfg.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		llmCfg.MaxTokens = *req.MaxTokens
	}
	h.App.Config.LLM = llmCfg

	if err := h.App.RebuildGateway(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.ConfigPath != "" {
		_ = config.PersistLLM(h.ConfigPath, h.App.Config.LLM)
	}
	if len(req.Pinned) > 0 {
		_ = savePinnedSpecs(h.App.Config.OutputDir, req.Pinned)
	}

	info, err := h.buildSettingsInfo("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "settings": info})
}

func (h *Handler) buildSettingsInfo(userID string) (map[string]any, error) {
	effective := h.effectiveLLMConfig(userID)
	provider := "geegoo"
	model := defaultModel
	if p := strings.TrimSpace(effective.Provider); p != "" {
		provider = p
	}
	if m := strings.TrimSpace(effective.Model); m != "" {
		model = m
	}
	if userID == "" && h.App != nil && h.App.Gateway != nil {
		if m := strings.TrimSpace(h.App.Gateway.Model()); m != "" {
			model = m
		}
	} else if h.App != nil {
		if gw, _, err := h.App.BuildGatewayFromLLMConfig(effective, false); err == nil && gw != nil {
			if m := strings.TrimSpace(gw.Model()); m != "" {
				model = m
			}
		}
	}

	provName := llm.ProviderName(provider)
	if provName == "" {
		provName = llm.ProviderDeepSeek
	}
	thinkingState := thinkingStateFromConfig(effective)
	thinkingOn := llm.ResolveThinkingEnabled(provName, model, effective.Thinking)

	var userDoc *userLLMSettings
	if userID != "" {
		userDoc, _ = loadUserLLMSettings(h.userSettingsOutputDir(), userID)
	}
	pinned := pinnedFromUserSettings(userDoc, provider, model)
	if len(pinned) == 0 {
		pinned = loadPinnedSpecs("")
		if h.App != nil && h.App.Config != nil {
			pinned = loadPinnedSpecs(h.App.Config.OutputDir)
		}
	}
	if len(pinned) == 0 {
		pinned = []map[string]any{{"provider": provider, "model": model, "default": true}}
	}

	providers := make([]map[string]any, 0, len(llm.Presets))
	for name, preset := range llm.Presets {
		providers = append(providers, map[string]any{
			"name": name, "label": preset.Label,
			"default_model": preset.DefaultModel,
		})
	}

	catalog := []map[string]any{}
	if models, err := h.fetchCatalogModels(context.Background()); err == nil {
		for _, m := range models {
			catalog = append(catalog, map[string]any{
				"model_id": m.ModelID, "name": m.Name, "display_name": m.DisplayName,
				"label": llm.CatalogModelLabel(m), "type": m.Type, "provider": m.Provider,
				"configured": m.Type == "configured",
			})
		}
	}

	temp := 0.7
	maxTok := 4096
	useOps := true
	catalogID := ""
	if effective.Temperature > 0 {
		temp = effective.Temperature
	}
	if effective.MaxTokens > 0 {
		maxTok = effective.MaxTokens
	}
	if effective.UseOpsModel != nil {
		useOps = *effective.UseOpsModel
	}
	catalogID = effective.CatalogModelID

	return map[string]any{
		"provider": provider, "model": model,
		"small_model": model,
		"thinking": thinkingState, "thinking_active": thinkingOn,
		"thinking_supported": llm.ModelSupportsThinking(provName, model),
		"temperature": temp, "max_tokens": maxTok,
		"use_ops_model": useOps, "catalog_model_id": catalogID,
		"pinned": pinned, "providers": providers, "catalog": catalog,
	}, nil
}

func (h *Handler) fetchCatalogModels(ctx context.Context) ([]admin.ConfiguredModel, error) {
	if h.App == nil || h.App.Config == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	targets := make([]admin.QueryTarget, 0, len(h.App.Config.AdminModelQueryTargets()))
	for _, t := range h.App.Config.AdminModelQueryTargets() {
		targets = append(targets, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
	}
	docs, _, err := admin.ListModelsFromTargets(ctx, targets)
	return docs, err
}

func pinnedPath(outputDir string) string {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = "."
	}
	return filepath.Join(outputDir, "dashboard_pins.json")
}

func loadPinnedSpecs(outputDir string) []map[string]any {
	path := pinnedPath(outputDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Pinned []string `json:"pinned"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(doc.Pinned))
	seen := map[string]bool{}
	for _, spec := range doc.Pinned {
		p, m, ok := strings.Cut(strings.TrimSpace(spec), ":")
		if !ok || m == "" {
			continue
		}
		out = append(out, map[string]any{
			"provider": p, "model": m, "default": !seen[p],
		})
		seen[p] = true
	}
	return out
}

func savePinnedSpecs(outputDir string, specs []string) error {
	path := pinnedPath(outputDir)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	raw, _ := json.MarshalIndent(map[string]any{"pinned": specs}, "", "  ")
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
