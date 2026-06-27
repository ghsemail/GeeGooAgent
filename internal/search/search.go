package search

import (
	"context"
	"strings"
)

// Provider name constants.
const (
	ProviderDuckDuckGo = "duckduckgo"
	ProviderOff        = "off"
)

// Config controls optional web search.
type Config struct {
	Provider   string
	MaxResults int
}

// DefaultConfig enables free DuckDuckGo search.
func DefaultConfig() Config {
	return Config{Provider: ProviderDuckDuckGo, MaxResults: 5}
}

// Search runs the configured provider.
func Search(ctx context.Context, cfg Config, query string) ([]Hit, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = ProviderDuckDuckGo
	}
	if provider == ProviderOff {
		return nil, nil
	}
	max := cfg.MaxResults
	if max <= 0 {
		max = 5
	}
	switch provider {
	case ProviderDuckDuckGo:
		return DuckDuckGo(ctx, query, max)
	default:
		return nil, nil
	}
}
