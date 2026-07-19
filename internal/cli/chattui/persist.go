package chattui

import "github.com/ghsemail/GeeGooAgent/internal/config"

// PersistDisplay merges display into config.json (delegates to config package).
func PersistDisplay(configPath string, display config.DisplayConfig) error {
	return config.PersistDisplay(configPath, display)
}
