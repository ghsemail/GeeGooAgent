package strategydev

import (
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

var klinePatternDisplayEnabled atomic.Bool

func init() {
	klinePatternDisplayEnabled.Store(true)
	if v := strings.TrimSpace(os.Getenv("GEEGOO_KLINE_PATTERN_DISPLAY")); v != "" {
		klinePatternDisplayEnabled.Store(envTruthy(v))
	}
}

// ConfigureKlinePatternDisplayEnabled sets the runtime switch (typically from loaded AppConfig).
func ConfigureKlinePatternDisplayEnabled(enabled bool) {
	klinePatternDisplayEnabled.Store(enabled)
}

// KlinePatternDisplayEnabled reports whether probe pattern_hits are parsed and attached to triggers/episodes.
func KlinePatternDisplayEnabled() bool {
	return klinePatternDisplayEnabled.Load()
}

// SetKlinePatternDisplayEnabledForTest toggles the switch for the current test only.
func SetKlinePatternDisplayEnabledForTest(t testing.TB, enabled bool) {
	t.Helper()
	prev := KlinePatternDisplayEnabled()
	_ = os.Unsetenv("GEEGOO_KLINE_PATTERN_DISPLAY")
	ConfigureKlinePatternDisplayEnabled(enabled)
	t.Cleanup(func() { ConfigureKlinePatternDisplayEnabled(prev) })
}
