package runtimeapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var safeUserID = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type userLLMSettings struct {
	Provider       string                        `json:"provider,omitempty" bson:"provider,omitempty"`
	Model          string                        `json:"model,omitempty" bson:"model,omitempty"`
	CatalogModelID string                        `json:"catalog_model_id,omitempty" bson:"catalog_model_id,omitempty"`
	UseOpsModel    *bool                         `json:"use_ops_model,omitempty" bson:"use_ops_model,omitempty"`
	Thinking       string                        `json:"thinking,omitempty" bson:"thinking,omitempty"` // on | off | auto
	Temperature    *float64                      `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens      *int                          `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	Pinned         []string                      `json:"pinned,omitempty" bson:"pinned,omitempty"`
	Gateways       map[string]gatewayLLMSettings `json:"gateways,omitempty" bson:"gateways,omitempty"`
	UpdatedAt      string                        `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

type gatewayLLMSettings struct {
	Provider       string `json:"provider,omitempty" bson:"provider,omitempty"`
	Model          string `json:"model,omitempty" bson:"model,omitempty"`
	CatalogModelID string `json:"catalog_model_id,omitempty" bson:"catalog_model_id,omitempty"`
	UseOpsModel    *bool  `json:"use_ops_model,omitempty" bson:"use_ops_model,omitempty"`
}

func userLLMSettingsPath(outputDir, userID string) string {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = "."
	}
	safe := safeUserID.ReplaceAllString(strings.TrimSpace(userID), "_")
	if safe == "" {
		safe = "anonymous"
	}
	return filepath.Join(outputDir, "user_llm_settings", safe+".json")
}

func loadUserLLMSettings(outputDir, userID string) (*userLLMSettings, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(userLLMSettingsPath(outputDir, userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc userLLMSettings
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func saveUserLLMSettings(outputDir, userID string, doc *userLLMSettings) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || doc == nil {
		return nil
	}
	doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	path := userLLMSettingsPath(outputDir, userID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func (u *userLLMSettings) applyRequest(req dashboardSettingsRequest) {
	if u == nil {
		return
	}
	gwKey := NormalizeSessionSource(req.Gateway)
	if strings.TrimSpace(req.Gateway) != "" && gwKey != "" {
		if u.Gateways == nil {
			u.Gateways = map[string]gatewayLLMSettings{}
		}
		entry := u.Gateways[gwKey]
		entry.applyRequest(req)
		u.Gateways[gwKey] = entry
		return
	}
	if p := strings.TrimSpace(req.Provider); p != "" {
		u.Provider = p
	}
	if m := strings.TrimSpace(req.Model); m != "" {
		u.Model = m
	}
	if id := strings.TrimSpace(req.CatalogModelID); id != "" {
		u.CatalogModelID = id
		useOps := false
		u.UseOpsModel = &useOps
	} else if req.UseOpsModel != nil {
		u.UseOpsModel = req.UseOpsModel
		if *req.UseOpsModel {
			u.CatalogModelID = ""
		}
	}
	if t := strings.ToLower(strings.TrimSpace(req.Thinking)); t != "" {
		u.Thinking = t
	}
	if req.Temperature != nil && *req.Temperature > 0 {
		u.Temperature = req.Temperature
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		u.MaxTokens = req.MaxTokens
	}
	if len(req.Pinned) > 0 {
		u.Pinned = append([]string(nil), req.Pinned...)
	}
}

func (g *gatewayLLMSettings) applyRequest(req dashboardSettingsRequest) {
	if g == nil {
		return
	}
	if p := strings.TrimSpace(req.Provider); p != "" {
		g.Provider = p
	}
	if m := strings.TrimSpace(req.Model); m != "" {
		g.Model = m
	}
	if id := strings.TrimSpace(req.CatalogModelID); id != "" {
		g.CatalogModelID = id
		useOps := false
		g.UseOpsModel = &useOps
	} else if req.UseOpsModel != nil {
		g.UseOpsModel = req.UseOpsModel
		if *req.UseOpsModel {
			g.CatalogModelID = ""
		}
	}
}

func (g gatewayLLMSettings) mergeInto(base config.LLMConfig) config.LLMConfig {
	out := base
	if p := strings.TrimSpace(g.Provider); p != "" {
		out.Provider = p
	}
	if m := strings.TrimSpace(g.Model); m != "" {
		out.Model = m
	}
	if id := strings.TrimSpace(g.CatalogModelID); id != "" {
		out.CatalogModelID = id
	} else if g.UseOpsModel != nil && *g.UseOpsModel {
		out.CatalogModelID = ""
	}
	if g.UseOpsModel != nil {
		out.UseOpsModel = g.UseOpsModel
	}
	return out
}

func (u *userLLMSettings) mergeInto(base config.LLMConfig) config.LLMConfig {
	if u == nil {
		return base
	}
	out := base
	if p := strings.TrimSpace(u.Provider); p != "" {
		out.Provider = p
	}
	if m := strings.TrimSpace(u.Model); m != "" {
		out.Model = m
	}
	if id := strings.TrimSpace(u.CatalogModelID); id != "" {
		out.CatalogModelID = id
	} else if u.UseOpsModel != nil && *u.UseOpsModel {
		out.CatalogModelID = ""
	}
	if u.UseOpsModel != nil {
		out.UseOpsModel = u.UseOpsModel
	}
	switch strings.ToLower(strings.TrimSpace(u.Thinking)) {
	case "on":
		v := true
		out.Thinking = &v
	case "off":
		v := false
		out.Thinking = &v
	case "auto":
		out.Thinking = nil
	}
	if u.Temperature != nil && *u.Temperature > 0 {
		out.Temperature = *u.Temperature
	}
	if u.MaxTokens != nil && *u.MaxTokens > 0 {
		out.MaxTokens = *u.MaxTokens
	}
	return out
}

func thinkingStateFromConfig(cfg config.LLMConfig) string {
	if cfg.Thinking == nil {
		return "auto"
	}
	if *cfg.Thinking {
		return "on"
	}
	return "off"
}

func pinnedFromUserSettings(u *userLLMSettings, provider, model string) []map[string]any {
	if u != nil && len(u.Pinned) > 0 {
		return pinnedSpecsToMaps(u.Pinned)
	}
	return nil
}

func pinnedSpecsToMaps(specs []string) []map[string]any {
	out := make([]map[string]any, 0, len(specs))
	seen := map[string]bool{}
	for _, spec := range specs {
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

func (h *Handler) userSettingsOutputDir() string {
	if h.App != nil && h.App.Config != nil && strings.TrimSpace(h.App.Config.OutputDir) != "" {
		return h.App.Config.OutputDir
	}
	if h.App != nil && strings.TrimSpace(h.App.Workspace) != "" {
		return h.App.Workspace
	}
	return "."
}

func (h *Handler) effectiveLLMConfig(userID, gateway string) config.LLMConfig {
	if h.App == nil || h.App.Config == nil {
		return config.LLMConfig{}
	}
	base := h.App.Config.LLM
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return base
	}
	us, err := h.loadUserLLMSettings(userID)
	if err != nil || us == nil {
		return base
	}
	out := us.mergeInto(base)
	gwKey := NormalizeSessionSource(gateway)
	if gwKey != "" && us.Gateways != nil {
		if gw, ok := us.Gateways[gwKey]; ok {
			out = gw.mergeInto(out)
		}
	}
	return out
}

func (h *Handler) withUserAgentGateway(userID, gateway string, fn func()) {
	userID = strings.TrimSpace(userID)
	if userID == "" || h.App == nil || h.App.Agent == nil {
		fn()
		return
	}
	cfg := h.effectiveLLMConfig(userID, gateway)
	gw, _, err := h.App.BuildGatewayFromLLMConfig(cfg, false)
	if err != nil || gw == nil {
		fn()
		return
	}
	h.gatewayMu.Lock()
	defer h.gatewayMu.Unlock()
	prev := h.App.Agent.Gateway
	h.App.Agent.SetGateway(gw)
	defer h.App.Agent.SetGateway(prev)
	fn()
}

func (h *Handler) userGateway(userID, gateway string) *llm.Gateway {
	if h.App == nil {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return h.App.Gateway
	}
	cfg := h.effectiveLLMConfig(userID, gateway)
	gw, _, err := h.App.BuildGatewayFromLLMConfig(cfg, false)
	if err != nil || gw == nil {
		return h.App.Gateway
	}
	return gw
}

func (h *Handler) gatewayForCompareSpec(userID, gateway, spec string) *llm.Gateway {
	if h.App == nil {
		return nil
	}
	provider, model := splitModelSpec(spec)
	cfg := h.effectiveLLMConfig(userID, gateway)
	cfg.Provider = provider
	cfg.Model = model
	cfg.CatalogModelID = ""
	useOps := false
	cfg.UseOpsModel = &useOps
	gw, _, err := h.App.BuildGatewayFromLLMConfig(cfg, false)
	if err != nil || gw == nil {
		return h.userGateway(userID, gateway)
	}
	return gw
}

func modelLabelFromConfig(cfg config.LLMConfig) string {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	provName := llm.ProviderName(cfg.Provider)
	if provName == "" {
		provName = llm.ProviderDeepSeek
	}
	return llm.ResolveModel(provName, model)
}
