package config

import (
	"encoding/json"
	"os"
)

// PersistChatToolsets merges chat_toolsets into config.json without clobbering other keys.
// Pass nil or empty slice to remove the key (revert to runtime defaults).
func PersistChatToolsets(configPath string, ids []string) error {
	if configPath == "" {
		return &ConfigError{Message: "empty config path"}
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if len(ids) == 0 {
		delete(doc, "chat_toolsets")
	} else {
		arr := make([]any, len(ids))
		for i, id := range ids {
			arr[i] = id
		}
		doc["chat_toolsets"] = arr
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(out, '\n'), 0o600)
}
