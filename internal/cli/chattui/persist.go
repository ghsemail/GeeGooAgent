package chattui

import (
	"encoding/json"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// PersistDisplay merges display into existing config.json without clobbering other keys.
func PersistDisplay(configPath string, display config.DisplayConfig) error {
	display.Normalize()
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	doc["display"] = display
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(out, '\n'), 0o600)
}
