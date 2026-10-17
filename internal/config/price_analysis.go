package config

import (
	"os"
	"strings"
)

// PriceAnalysisEnabled is the master switch for strategy-dev / signal-diagnose price analysis.
// JSON: price_analysis_enabled (default false when unset).
// Env: GEEGOO_PRICE_ANALYSIS_ENABLED overrides config when set.
func (c *AppConfig) PriceAnalysisEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_PRICE_ANALYSIS_ENABLED")); v != "" {
		return envTruthy(v)
	}
	if c != nil && c.PriceAnalysisEnabledField != nil {
		return *c.PriceAnalysisEnabledField
	}
	return false
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
