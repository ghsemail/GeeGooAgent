package strategydev

import (
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

var priceAnalysisFeatureEnabled atomic.Bool

func init() {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_PRICE_ANALYSIS_ENABLED")); v != "" {
		priceAnalysisFeatureEnabled.Store(envTruthy(v))
	}
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ConfigurePriceAnalysisEnabled sets the runtime master switch (typically from loaded AppConfig).
func ConfigurePriceAnalysisEnabled(enabled bool) {
	priceAnalysisFeatureEnabled.Store(enabled)
}

// PriceAnalysisFeatureEnabled reports whether multi-TF price analysis paths are active.
func PriceAnalysisFeatureEnabled() bool {
	return priceAnalysisFeatureEnabled.Load()
}

// SetPriceAnalysisFeatureEnabledForTest toggles the switch for the current test only.
func SetPriceAnalysisFeatureEnabledForTest(t testing.TB, enabled bool) {
	t.Helper()
	prev := PriceAnalysisFeatureEnabled()
	_ = os.Unsetenv("GEEGOO_PRICE_ANALYSIS_ENABLED")
	ConfigurePriceAnalysisEnabled(enabled)
	t.Cleanup(func() { ConfigurePriceAnalysisEnabled(prev) })
}
