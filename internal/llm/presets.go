package llm

import (
	"fmt"
	"strconv"
	"strings"
)

// ProviderName identifies built-in LLM backends.
type ProviderName string

const (
	ProviderOpenAI   ProviderName = "openai"
	ProviderDeepSeek ProviderName = "deepseek"
	ProviderMinimax  ProviderName = "minimax"
)

// Preset holds default base URL and model for a provider.
type Preset struct {
	Name         ProviderName
	Label        string
	BaseURL      string
	DefaultModel string
}

// Presets maps provider name to preset.
var Presets = map[ProviderName]Preset{
	ProviderOpenAI: {
		Name:         ProviderOpenAI,
		Label:        "OpenAI",
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-4o",
	},
	ProviderDeepSeek: {
		Name:         ProviderDeepSeek,
		Label:        "DeepSeek",
		BaseURL:      "https://api.deepseek.com",
		DefaultModel: "deepseek-v4-flash",
	},
	ProviderMinimax: {
		Name:         ProviderMinimax,
		Label:        "Minimax",
		BaseURL:      "https://api.minimaxi.com/v1",
		DefaultModel: "MiniMax-M2.1",
	},
}

// ProviderModel is one selectable model for /model.
type ProviderModel struct {
	ID          string
	Description string
}

// ProviderModels lists known models per provider (Python PROVIDER_MODELS).
var ProviderModels = map[ProviderName][]ProviderModel{
	ProviderDeepSeek: {
		{ID: "deepseek-v4-flash", Description: "V4 Flash，快速对话（推荐默认）"},
		{ID: "deepseek-v4-pro", Description: "V4 Pro，复杂推理 / 思考模式"},
		{ID: "deepseek-chat", Description: "旧版 chat（2026/07 弃用，兼容）"},
		{ID: "deepseek-reasoner", Description: "旧版 reasoner（2026/07 弃用，兼容）"},
	},
	ProviderOpenAI: {
		{ID: "gpt-4o", Description: "GPT-4o"},
		{ID: "gpt-4o-mini", Description: "GPT-4o mini"},
		{ID: "gpt-4-turbo", Description: "GPT-4 Turbo"},
	},
	ProviderMinimax: {
		{ID: "MiniMax-M2.1", Description: "MiniMax M2.1"},
	},
}

// ListProviderModels returns models for a provider.
func ListProviderModels(name ProviderName) []ProviderModel {
	return ProviderModels[name]
}

// ResolveModel returns configured model or provider default.
func ResolveModel(name ProviderName, model string) string {
	if strings.TrimSpace(model) != "" {
		return strings.TrimSpace(model)
	}
	if p, ok := Presets[name]; ok {
		return p.DefaultModel
	}
	return "gpt-4o"
}

// PickModel resolves /model selection: 1-based index, model id, or default.
func PickModel(name ProviderName, choice, current string) (string, error) {
	text := strings.TrimSpace(choice)
	if text == "" {
		return ResolveModel(name, current), nil
	}
	models := ListProviderModels(name)
	if n, err := strconv.Atoi(text); err == nil {
		if n >= 1 && n <= len(models) {
			return models[n-1].ID, nil
		}
		return "", fmt.Errorf("invalid model index: %d (1-%d)", n, len(models))
	}
	if len(models) > 0 {
		known := map[string]struct{}{}
		for _, m := range models {
			known[m.ID] = struct{}{}
		}
		if _, ok := known[text]; !ok {
			return "", fmt.Errorf("unknown model for %s: %s", name, text)
		}
	}
	return text, nil
}

// ModelSupportsThinking reports whether thinking mode applies.
func ModelSupportsThinking(name ProviderName, model string) bool {
	if name != ProviderDeepSeek {
		return false
	}
	resolved := strings.ToLower(ResolveModel(name, model))
	return strings.Contains(resolved, "v4") || resolved == "deepseek-reasoner"
}

// ResolveThinkingEnabled mirrors Python resolve_thinking_enabled.
func ResolveThinkingEnabled(name ProviderName, model string, thinking *bool) bool {
	if !ModelSupportsThinking(name, model) {
		return false
	}
	if thinking == nil {
		return strings.Contains(strings.ToLower(ResolveModel(name, model)), "v4")
	}
	return *thinking
}
