package config

import (
	"encoding/json"
	"os"
)

// PersistLLM merges llm fields into config.json without clobbering other keys.
func PersistLLM(configPath string, llm LLMConfig) error {
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
	llmRaw, _ := doc["llm"].(map[string]any)
	if llmRaw == nil {
		llmRaw = map[string]any{}
		doc["llm"] = llmRaw
	}
	if llm.Provider != "" {
		llmRaw["provider"] = llm.Provider
	}
	if llm.TokenKey != "" {
		llmRaw["token_key"] = llm.TokenKey
	}
	if llm.Model != "" {
		llmRaw["model"] = llm.Model
	}
	if llm.BaseURL != "" {
		llmRaw["base_url"] = llm.BaseURL
	} else {
		delete(llmRaw, "base_url")
	}
	if llm.CatalogModelID != "" {
		llmRaw["catalog_model_id"] = llm.CatalogModelID
	} else {
		delete(llmRaw, "catalog_model_id")
	}
	if llm.UseOpsModel != nil {
		llmRaw["use_ops_model"] = *llm.UseOpsModel
	}
	if llm.Thinking == nil {
		delete(llmRaw, "thinking")
	} else {
		llmRaw["thinking"] = *llm.Thinking
	}
	if llm.Temperature > 0 {
		llmRaw["temperature"] = llm.Temperature
	}
	if llm.MaxTokens > 0 {
		llmRaw["max_tokens"] = llm.MaxTokens
	}
	if llm.ReasoningEffort != "" {
		llmRaw["reasoning_effort"] = llm.ReasoningEffort
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(out, '\n'), 0o600)
}
