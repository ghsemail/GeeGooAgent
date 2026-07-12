package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Save writes cfg as indented JSON to path.
func Save(path string, cfg *AppConfig) error {
	if cfg == nil {
		return &ConfigError{Message: "nil config"}
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return &ConfigError{Message: fmt.Sprintf("marshal config: %v", err)}
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		return &ConfigError{Message: fmt.Sprintf("write config: %v", err)}
	}
	return nil
}
