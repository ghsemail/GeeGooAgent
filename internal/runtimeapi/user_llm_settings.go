package runtimeapi

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/userllmstore"
)

type userLLMSettings = userllmstore.Settings
type gatewayLLMSettings = userllmstore.GatewayEntry

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

func (h *Handler) loadUserLLMSettings(userID string) (*userLLMSettings, error) {
	if h.App == nil || h.App.UserLLM == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return h.App.UserLLM.Load(ctx, userID)
}

func (h *Handler) saveUserLLMSettings(userID string, doc *userLLMSettings) error {
	if h.App == nil || h.App.UserLLM == nil || doc == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return h.App.UserLLM.Save(ctx, userID, doc)
}

func (h *Handler) effectiveLLMConfig(userID, gateway string) config.LLMConfig {
	if h.App == nil {
		return config.LLMConfig{}
	}
	return h.App.EffectiveLLMConfig(userID, NormalizeSessionSource(gateway))
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
	prev := h.App.Agent.Gateway
	h.App.Agent.SetGateway(gw)
	h.gatewayMu.Unlock()
	defer func() {
		h.gatewayMu.Lock()
		h.App.Agent.SetGateway(prev)
		h.gatewayMu.Unlock()
	}()
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

func userLLMSettingsPath(outputDir, userID string) string {
	return userllmstore.SettingsFilePath(outputDir, userID)
}

func loadUserLLMSettings(outputDir, userID string) (*userLLMSettings, error) {
	backend := userllmstore.NewBackend(nil, outputDir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return backend.Load(ctx, userID)
}

func saveUserLLMSettings(outputDir, userID string, doc *userLLMSettings) error {
	backend := userllmstore.NewBackend(nil, outputDir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return backend.Save(ctx, userID, doc)
}
