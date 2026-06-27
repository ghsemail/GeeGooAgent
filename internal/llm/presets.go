package llm

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

// ResolveModel returns configured model or provider default.
func ResolveModel(name ProviderName, model string) string {
	if model != "" {
		return model
	}
	if p, ok := Presets[name]; ok {
		return p.DefaultModel
	}
	return "gpt-4o"
}
