package chatprompt

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const soulFileName = "SOUL.md"

// SoulMaxBytes is the maximum SOUL.md size accepted from dashboard edits.
const SoulMaxBytes = 64 * 1024

// SoulPath returns the editable persona file path under GEEGOO_HOME.
func SoulPath(home string) string {
	if strings.TrimSpace(home) == "" {
		home = config.Home()
	}
	return filepath.Join(home, soulFileName)
}

// DefaultSoul returns the built-in persona when SOUL.md is missing.
func DefaultSoul() string {
	return defaultSoulText
}

// LoadSoulFromHome reads SOUL.md from home; missing file returns DefaultSoul().
func LoadSoulFromHome(home string) string {
	path := SoulPath(home)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSoul()
		}
		return DefaultSoul()
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return DefaultSoul()
	}
	return strings.TrimRight(string(b), "\n") + "\n"
}

// SaveSoulToHome writes SOUL.md under home, creating the directory if needed.
func SaveSoulToHome(home, content string) error {
	text := strings.TrimSpace(content)
	if text == "" {
		return errSoulEmpty
	}
	if len([]byte(text)) > SoulMaxBytes {
		return errSoulTooLarge
	}
	path := SoulPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := strings.TrimRight(text, "\n") + "\n"
	return os.WriteFile(path, []byte(payload), 0o644)
}
