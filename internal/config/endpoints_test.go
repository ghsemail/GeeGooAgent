package config_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestGeeGooEndpointDefaults(t *testing.T) {
	cfg := &config.AppConfig{}
	if cfg.EffectiveMCPURL() != config.DefaultBotMCPURL {
		t.Fatalf("MCP default = %q", cfg.EffectiveMCPURL())
	}
	if cfg.SignalCatalogURL() != config.DefaultSignalCatalogURL {
		t.Fatalf("catalog default = %q", cfg.SignalCatalogURL())
	}
	if cfg.DataHTTPURL() != config.DefaultDataHTTPURL {
		t.Fatalf("data default = %q", cfg.DataHTTPURL())
	}
}

func TestLegacyPortWarning(t *testing.T) {
	cfg := &config.AppConfig{
		GeeGooURL:     "http://118.195.135.97:5700",
		SignalBaseURL: "http://146.56.225.252:5800",
	}
	warnings := cfg.LegacyPortWarnings()
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestAdminModelQueryTargets(t *testing.T) {
	cfg := &config.AppConfig{
		SignalBaseURL:            "http://146.56.225.252:3210",
		SignalCatalogAPIKeyField: "cat-key",
	}
	targets := cfg.AdminModelQueryTargets()
	if len(targets) != 1 {
		t.Fatalf("expected catalog only, got %v", targets)
	}
	if targets[0].BaseURL != "http://146.56.225.252:3210" || targets[0].Bearer != "cat-key" {
		t.Fatalf("catalog target = %+v", targets[0])
	}
}

func TestSignalAPIURLFromCatalogHost(t *testing.T) {
	cfg := &config.AppConfig{SignalBaseURL: "http://146.56.225.252:3210"}
	if got := cfg.SignalAPIURL(); got != "http://146.56.225.252:3200" {
		t.Fatalf("signal api = %q", got)
	}
}

func TestEnvOverridesSignalAndData(t *testing.T) {
	t.Setenv("GEEGOO_SIGNAL_CATALOG_API_URL", "http://signal.local:3210")
	t.Setenv("GEEGOO_DATA_HTTP_URL", "http://data.local:3300")
	cfg := &config.AppConfig{}
	if cfg.SignalCatalogURL() != "http://signal.local:3210" {
		t.Fatalf("catalog = %q", cfg.SignalCatalogURL())
	}
	if cfg.DataHTTPURL() != "http://data.local:3300" {
		t.Fatalf("data = %q", cfg.DataHTTPURL())
	}
}
