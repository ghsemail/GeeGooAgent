package config

import (
	"os"
	"strings"
)

const DefaultSignalLabAPIURL = "http://127.0.0.1:3250"

// LabEnabled is the master switch for Agent /v1/lab/* routes (default false).
// JSON: lab_enabled. Env: GEEGOO_AGENT_LAB_ENABLED overrides when set.
func (c *AppConfig) LabEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_AGENT_LAB_ENABLED")); v != "" {
		return envTruthy(v)
	}
	if c != nil && c.LabEnabledField != nil {
		return *c.LabEnabledField
	}
	return false
}

// LabAPIURL returns GeeGooSignal lab-api base URL.
func (c *AppConfig) LabAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_AGENT_LAB_API_URL")); v != "" {
		return trimSlash(v)
	}
	if c != nil && strings.TrimSpace(c.LabAPIURLField) != "" {
		return trimSlash(c.LabAPIURLField)
	}
	if c != nil && c.SignalBaseURL != "" {
		if u := replacePort(c.SignalBaseURL, "3250"); u != "" {
			return u
		}
	}
	return DefaultSignalLabAPIURL
}

// LabAPIKey returns Bearer for GeeGooSignal lab-api.
func (c *AppConfig) LabAPIKey() string {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_SIGNAL_LAB_API_KEY")); v != "" {
		return v
	}
	if c != nil && strings.TrimSpace(c.LabAPIKeyField) != "" {
		return c.LabAPIKeyField
	}
	return c.SignalAPIKey()
}
