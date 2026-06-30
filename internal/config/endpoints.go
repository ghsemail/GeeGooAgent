package config

import (
	"fmt"
	"os"
	"strings"
)

// GeeGoo 服务默认端口（见 docs/refactor/ports.md）。
const (
	DefaultBotMCPURL        = "http://127.0.0.1:3120"
	DefaultSignalCatalogURL = "http://127.0.0.1:3210"
	DefaultSignalAnalyzeURL = "http://127.0.0.1:3230"
	DefaultDataHTTPURL      = "http://127.0.0.1:3300"
)

// SignalCatalogURL returns GeeGooSignal catalog-api (:3210).
func (c *AppConfig) SignalCatalogURL() string {
	if v := os.Getenv("GEEGOO_SIGNAL_CATALOG_API_URL"); v != "" {
		return trimSlash(v)
	}
	if c.SignalBaseURL != "" {
		return trimSlash(c.SignalBaseURL)
	}
	return DefaultSignalCatalogURL
}

// SignalAnalyzeURL returns GeeGooSignal analyze-api (:3230).
func (c *AppConfig) SignalAnalyzeURL() string {
	if v := os.Getenv("GEEGOO_SIGNAL_ANALYZE_API_URL"); v != "" {
		return trimSlash(v)
	}
	return DefaultSignalAnalyzeURL
}

// DataHTTPURL returns GeeGooData data-api (:3300).
func (c *AppConfig) DataHTTPURL() string {
	if v := os.Getenv("GEEGOO_DATA_HTTP_URL"); v != "" {
		return trimSlash(v)
	}
	if c.DataBaseURL != "" {
		return trimSlash(c.DataBaseURL)
	}
	return DefaultDataHTTPURL
}

// EffectiveMCPURL returns MCP URL with default fallback.
func (c *AppConfig) EffectiveMCPURL() string {
	if u := c.MCPURL(); u != "" {
		return u
	}
	return DefaultBotMCPURL
}

// LegacyPortWarnings detects Trading-era ports in config (5700/5800/5600).
func (c *AppConfig) LegacyPortWarnings() []string {
	var warnings []string
	for _, u := range []string{c.BaseURL, c.GeeGooURL, c.SignalBaseURL, c.DataBaseURL} {
		if hasLegacyPort(u) {
			warnings = append(warnings, fmt.Sprintf(
				"config URL %q uses legacy Trading port; prefer GeeGooBot :3120 / GeeGooSignal :3210 / GeeGooData :3300",
				u,
			))
		}
	}
	return warnings
}

func hasLegacyPort(raw string) bool {
	legacy := []string{":5700", ":5800", ":5600", ":5500", ":5900", ":6100", ":6200"}
	for _, p := range legacy {
		if strings.Contains(raw, p) {
			return true
		}
	}
	return false
}

// DefaultAllowedHosts for sandbox when config omits the list.
func DefaultAllowedHosts() []string {
	return []string{"127.0.0.1", "localhost"}
}

// ResolvedAllowedHosts returns sandbox hosts from config or defaults.
func (c *AppConfig) ResolvedAllowedHosts() []string {
	if len(c.Sandbox.AllowedHosts) > 0 {
		return c.Sandbox.AllowedHosts
	}
	return DefaultAllowedHosts()
}
