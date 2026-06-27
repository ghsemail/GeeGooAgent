package config

import (
	"os"
	"path/filepath"
)

// Home returns GEEGOO_HOME or ~/.geegoo.
func Home() string {
	if v := os.Getenv("GEEGOO_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".geegoo"
	}
	return filepath.Join(home, ".geegoo")
}

// DefaultPath resolves the config file (GEEGOO_CONFIG > ~/.geegoo/config.json > ./config.json).
func DefaultPath() string {
	if v := os.Getenv("GEEGOO_CONFIG"); v != "" {
		return v
	}
	homeCfg := filepath.Join(Home(), "config.json")
	if _, err := os.Stat(homeCfg); err == nil {
		return homeCfg
	}
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}
	return homeCfg
}
