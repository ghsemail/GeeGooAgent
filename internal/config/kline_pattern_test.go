package config

import "testing"

func TestKlinePatternDisplayEnabledDefaultFalse(t *testing.T) {
	t.Setenv("GEEGOO_KLINE_PATTERN_DISPLAY", "")
	if (&AppConfig{}).KlinePatternDisplayEnabled() {
		t.Fatal("expected default off")
	}
}

func TestKlinePatternDisplayEnabledEnvAndJSON(t *testing.T) {
	t.Setenv("GEEGOO_KLINE_PATTERN_DISPLAY", "true")
	if !(&AppConfig{}).KlinePatternDisplayEnabled() {
		t.Fatal("env should enable")
	}
	t.Setenv("GEEGOO_KLINE_PATTERN_DISPLAY", "")
	on := true
	if !(&AppConfig{KlinePatternDisplayField: &on}).KlinePatternDisplayEnabled() {
		t.Fatal("json field should enable when env unset")
	}
	off := false
	if (&AppConfig{KlinePatternDisplayField: &off}).KlinePatternDisplayEnabled() {
		t.Fatal("json false should disable when env unset")
	}
}
