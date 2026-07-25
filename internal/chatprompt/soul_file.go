package chatprompt

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const soulFileName = "SOUL.md"

var safeUserIDRe = regexp.MustCompile(`[^a-zA-Z0-9._@-]+`)

// SoulMaxBytes is the maximum SOUL.md size accepted from dashboard edits.
const SoulMaxBytes = 64 * 1024

// SanitizeUserID makes a user id safe for tenant directory names.
func SanitizeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	return safeUserIDRe.ReplaceAllString(userID, "_")
}

// SoulPath returns the editable persona file path under GEEGOO_HOME (global).
func SoulPath(home string) string {
	return SoulPathForUser(home, "")
}

// SoulPathForUser returns per-tenant SOUL.md when userID is set, else global SOUL.md.
func SoulPathForUser(home, userID string) string {
	if strings.TrimSpace(home) == "" {
		home = config.Home()
	}
	if uid := SanitizeUserID(userID); uid != "" {
		return filepath.Join(home, "tenants", uid, soulFileName)
	}
	return filepath.Join(home, soulFileName)
}

// DefaultSoul returns the built-in persona when SOUL.md is missing.
func DefaultSoul() string {
	return defaultSoulText
}

// LoadSoulFromHome reads global SOUL.md; missing file returns DefaultSoul().
func LoadSoulFromHome(home string) string {
	return LoadSoulForUser(home, "")
}

// LoadSoulForUser reads tenant SOUL, then global SOUL, then default.
func LoadSoulForUser(home, userID string) string {
	if text, ok := readSoulFile(SoulPathForUser(home, userID)); ok {
		return text
	}
	if uid := SanitizeUserID(userID); uid != "" {
		if text, ok := readSoulFile(SoulPath(home)); ok {
			return text
		}
	}
	return DefaultSoul()
}

func readSoulFile(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return "", false
	}
	return strings.TrimRight(string(b), "\n") + "\n", true
}

// SaveSoulToHome writes global SOUL.md under home.
func SaveSoulToHome(home, content string) error {
	return SaveSoulForUser(home, "", content)
}

// SaveSoulForUser writes tenant-scoped SOUL.md.
func SaveSoulForUser(home, userID, content string) error {
	text := strings.TrimSpace(content)
	if text == "" {
		return errSoulEmpty
	}
	if len([]byte(text)) > SoulMaxBytes {
		return errSoulTooLarge
	}
	path := SoulPathForUser(home, userID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := strings.TrimRight(text, "\n") + "\n"
	return os.WriteFile(path, []byte(payload), 0o644)
}
