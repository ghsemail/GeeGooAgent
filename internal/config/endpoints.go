package config

import (
	"fmt"
	"os"
	"strings"
)

// GeeGoo 服务默认端口（见 docs/refactor/ports.md）。
const (
	DefaultBotMCPURL        = "http://127.0.0.1:3120"
	DefaultSignalAPIURL     = "http://127.0.0.1:3200"
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

// SignalAPIURL returns GeeGooSignal signal-api (:3200).
func (c *AppConfig) SignalAPIURL() string {
	if v := os.Getenv("GEEGOO_SIGNAL_SIGNAL_API_URL"); v != "" {
		return trimSlash(v)
	}
	if c.SignalAPIURLField != "" {
		return trimSlash(c.SignalAPIURLField)
	}
	if c.SignalBaseURL != "" {
		if u := replacePort(c.SignalBaseURL, "3200"); u != "" {
			return u
		}
	}
	return DefaultSignalAPIURL
}

// SignalAPIKey returns Bearer for GeeGooSignal signal-api.
func (c *AppConfig) SignalAPIKey() string {
	if v := os.Getenv("GEEGOO_SIGNAL_SIGNAL_API_KEY"); v != "" {
		return v
	}
	if c.SignalAPIKeyField != "" {
		return c.SignalAPIKeyField
	}
	return c.MCPAPIKey()
}

// SignalCatalogAPIKey returns Bearer for GeeGooSignal catalog-api.
func (c *AppConfig) SignalCatalogAPIKey() string {
	if v := os.Getenv("GEEGOO_SIGNAL_CATALOG_API_KEY"); v != "" {
		return v
	}
	if c.SignalCatalogAPIKeyField != "" {
		return c.SignalCatalogAPIKeyField
	}
	return c.SignalAPIKey()
}

// DefaultPythonAdminURL is TradingSignal adminServer (ops LLM credentials until catalog-api returns token).
const DefaultPythonAdminURL = "http://146.56.225.252:5800"

// AdminModelURLs returns candidate bases for POST /queryModel (legacy helper).
func (c *AppConfig) AdminModelURLs() []string {
	out := make([]string, 0, len(c.AdminModelQueryTargets()))
	for _, t := range c.AdminModelQueryTargets() {
		out = append(out, t.BaseURL)
	}
	return out
}

// AdminModelQueryTarget pairs a queryModel base with optional Bearer.
type AdminModelQueryTarget struct {
	BaseURL string
	Bearer  string
}

// AdminModelQueryTargets lists ops model sources: catalog :3210 (Bearer), then Python admin :5800 fallback.
func (c *AppConfig) AdminModelQueryTargets() []AdminModelQueryTarget {
	var out []AdminModelQueryTarget
	if v := os.Getenv("GEEGOO_ADMIN_URL"); v != "" {
		out = append(out, AdminModelQueryTarget{BaseURL: trimSlash(v)})
	}
	out = append(out, AdminModelQueryTarget{
		BaseURL: c.SignalCatalogURL(),
		Bearer:  c.SignalCatalogAPIKey(),
	})
	catalog := c.SignalCatalogURL()
	if u := replacePort(catalog, "5800"); u != "" && u != catalog {
		out = append(out, AdminModelQueryTarget{BaseURL: u})
	}
	if !strings.Contains(catalog, ":5800") {
		out = append(out, AdminModelQueryTarget{BaseURL: DefaultPythonAdminURL})
	}
	return uniqueQueryTargets(out)
}

func uniqueQueryTargets(in []AdminModelQueryTarget) []AdminModelQueryTarget {
	seen := map[string]struct{}{}
	var out []AdminModelQueryTarget
	for _, t := range in {
		base := strings.TrimSpace(t.BaseURL)
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		out = append(out, AdminModelQueryTarget{BaseURL: base, Bearer: t.Bearer})
	}
	return out
}

func replacePort(raw, port string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// crude host:port swap for http(s)://host:port
	schemeSep := "://"
	i := strings.Index(raw, schemeSep)
	if i < 0 {
		return ""
	}
	rest := raw[i+len(schemeSep):]
	host := rest
	if j := strings.Index(rest, "/"); j >= 0 {
		host = rest[:j]
	}
	if k := strings.LastIndex(host, ":"); k >= 0 {
		host = host[:k]
	}
	return raw[:i+len(schemeSep)] + host + ":" + port
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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
