package config

import (
	"os"
	"strings"
)

const DefaultMacroAPIURL = "http://127.0.0.1:3220"

// MacroAPIURL returns GeeGooSignal macro-api (:3220) for diagnose fact lanes.
func (c *AppConfig) MacroAPIURL() string {
	if v := os.Getenv("GEEGOO_SIGNAL_MACRO_API_URL"); v != "" {
		return trimSlash(v)
	}
	if v := os.Getenv("MACRO_API_BASE"); v != "" {
		return trimSlash(v)
	}
	if c != nil && strings.TrimSpace(c.SignalBaseURL) != "" {
		if u := replacePort(c.SignalBaseURL, "3220"); u != "" {
			return u
		}
	}
	return DefaultMacroAPIURL
}

// MacroAPIKey returns Bearer for macro-api (/v1/diagnose/* and /macro/*).
func (c *AppConfig) MacroAPIKey() string {
	if v := os.Getenv("GEEGOO_SIGNAL_MACRO_API_KEY"); v != "" {
		return v
	}
	if v := os.Getenv("MACRO_API_KEY"); v != "" {
		return v
	}
	return c.SignalAPIKey()
}