package llm

import "strings"

// DefaultContextWindow is used when the model is unknown.
const DefaultContextWindow = 128_000

// knownContextWindows maps model id substrings / exact ids to context size.
// Prefer longer / more specific keys by checking exact match first, then prefix.
var knownContextWindows = map[string]int{
	// DeepSeek
	"deepseek-v4-pro":     128_000,
	"deepseek-v4-flash":   128_000,
	"deepseek-chat":       128_000,
	"deepseek-reasoner":   128_000,
	"deepseek":            128_000,
	// OpenAI family
	"gpt-4o-mini":         128_000,
	"gpt-4o":              128_000,
	"gpt-4-turbo":         128_000,
	"gpt-4.1":             1_047_576,
	"gpt-4":               128_000,
	"gpt-5.5":             400_000,
	"gpt-5.2":             400_000,
	"gpt-5.1":             400_000,
	"gpt-5":               400_000,
	"o3":                  200_000,
	"o1":                  200_000,
	// MiniMax
	"minimax-m2.1":        204_800,
	"minimax":             204_800,
	// Claude (if routed via OpenAI-compatible)
	"claude-opus-4":       200_000,
	"claude-sonnet-4":     200_000,
	"claude-3-5":          200_000,
	"claude-3":            200_000,
	"claude":              200_000,
}

// ResolveContextWindow returns the model context window.
// configured > 0 always wins (explicit config.compression.context_length).
func ResolveContextWindow(model string, configured int) int {
	if configured > 0 {
		return configured
	}
	name := strings.ToLower(strings.TrimSpace(model))
	if name == "" {
		return DefaultContextWindow
	}
	if n, ok := knownContextWindows[name]; ok {
		return n
	}
	// Longest prefix / contains match among known keys.
	bestKey := ""
	bestN := 0
	for key, n := range knownContextWindows {
		if strings.Contains(name, key) && len(key) > len(bestKey) {
			bestKey = key
			bestN = n
		}
	}
	if bestN > 0 {
		return bestN
	}
	return DefaultContextWindow
}
