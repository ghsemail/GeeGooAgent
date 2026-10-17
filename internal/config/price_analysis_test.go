package config

import (
	"testing"
)

func TestPriceAnalysisEnabledDefaultFalse(t *testing.T) {
	t.Setenv("GEEGOO_PRICE_ANALYSIS_ENABLED", "")
	if (&AppConfig{}).PriceAnalysisEnabled() {
		t.Fatal("expected default off")
	}
}

func TestPriceAnalysisEnabledEnvAndJSON(t *testing.T) {
	t.Setenv("GEEGOO_PRICE_ANALYSIS_ENABLED", "true")
	if !(&AppConfig{}).PriceAnalysisEnabled() {
		t.Fatal("env should enable")
	}
	t.Setenv("GEEGOO_PRICE_ANALYSIS_ENABLED", "")
	on := true
	if !(&AppConfig{PriceAnalysisEnabledField: &on}).PriceAnalysisEnabled() {
		t.Fatal("json field should enable when env unset")
	}
	off := false
	if (&AppConfig{PriceAnalysisEnabledField: &off}).PriceAnalysisEnabled() {
		t.Fatal("json false should disable when env unset")
	}
}
