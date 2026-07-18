package config

import (
	"fmt"
	"os"
	"strings"
)

// ProfileConfig holds per-profile overrides (Hermes profile isolation parity).
type ProfileConfig struct {
	OutputDir    string   `json:"output_dir,omitempty"`
	MCPToken     string   `json:"mcp_token,omitempty"`
	ChatToolsets []string `json:"chat_toolsets,omitempty"`
	DryRun       *bool    `json:"dry_run,omitempty"`
}

// ProfileFeatureEnabled reports whether profile switching is configured (optional feature).
func (c *AppConfig) ProfileFeatureEnabled() bool {
	if c == nil {
		return false
	}
	if len(c.Profiles) > 0 {
		return true
	}
	if strings.TrimSpace(os.Getenv("GEEGOO_PROFILE")) != "" {
		return true
	}
	return strings.TrimSpace(c.ActiveProfile) != ""
}

func (c *AppConfig) profileSource() string {
	if strings.TrimSpace(os.Getenv("GEEGOO_PROFILE")) != "" {
		return "GEEGOO_PROFILE"
	}
	if c != nil && strings.TrimSpace(c.ActiveProfile) != "" {
		return "active_profile"
	}
	return "default"
}

// ProfileOverridesApplied reports whether config.profiles contains the resolved name.
func (c *AppConfig) ProfileOverridesApplied() bool {
	if c == nil || len(c.Profiles) == 0 {
		return false
	}
	_, ok := c.Profiles[c.ResolvedProfile]
	return ok
}

// ProfileSummary is a one-line diagnostic for doctor / verify.
func (c *AppConfig) ProfileSummary() string {
	if c == nil {
		return "default"
	}
	name := c.ResolvedProfile
	if name == "" {
		name = "default"
	}
	src := c.profileSource()
	var parts []string
	if c.ProfileOverridesApplied() {
		if c.OutputDir != "" {
			parts = append(parts, "output_dir="+c.OutputDir)
		}
		if len(c.ChatToolsets) > 0 {
			parts = append(parts, "chat_toolsets="+strings.Join(c.ChatToolsets, ","))
		}
		if c.DryRun {
			parts = append(parts, "dry_run=true")
		}
		if c.MCPToken() != "" {
			parts = append(parts, "mcp_token=set")
		}
	} else if len(c.Profiles) > 0 && name != "default" {
		parts = append(parts, "no overrides (profile undefined)")
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s via %s", name, src)
	}
	return fmt.Sprintf("%s via %s (%s)", name, src, strings.Join(parts, ", "))
}
