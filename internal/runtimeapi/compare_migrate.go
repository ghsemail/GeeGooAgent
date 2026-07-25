package runtimeapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MigrateLegacyCompareHistory copies a global compare/history.jsonl into a user-scoped file.
func MigrateLegacyCompareHistory(workspace, userID, legacyPath string) (copied int, err error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, nil
	}
	if legacyPath == "" {
		legacyPath = filepath.Join(workspace, "compare", "history.jsonl")
	}
	raw, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	dest := filepath.Join(workspace, "compare", safeUserID.ReplaceAllString(userID, "_"), "history.jsonl")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	lines := strings.Split(string(raw), "\n")
	out := []byte{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var run compareRun
		if json.Unmarshal([]byte(line), &run) != nil {
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
		copied++
	}
	if len(out) == 0 {
		return 0, nil
	}
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		return 0, err
	}
	return copied, nil
}
