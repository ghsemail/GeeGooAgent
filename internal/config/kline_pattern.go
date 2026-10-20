package config

import (
	"os"
	"strings"
)

// KlinePatternDisplayEnabled controls optional K-line pattern annotations on diagnose triggers.
// JSON: kline_pattern_display (default false when unset).
// Env: GEEGOO_KLINE_PATTERN_DISPLAY overrides config when set.
func (c *AppConfig) KlinePatternDisplayEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_KLINE_PATTERN_DISPLAY")); v != "" {
		return envTruthy(v)
	}
	if c != nil && c.KlinePatternDisplayField != nil {
		return *c.KlinePatternDisplayField
	}
	return false
}
