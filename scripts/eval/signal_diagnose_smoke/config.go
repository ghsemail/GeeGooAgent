package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/weknora"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func optionalWeKnora() *weknora.Client {
	path := strings.TrimSpace(os.Getenv("GEEGOO_CONFIG"))
	if path == "" {
		if key := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_API_KEY")); key != "" {
			resolved := config.ResolvedWeKnora{
				APIURL: envOr("GEEGOO_WEKNORA_API_URL", config.DefaultWeKnoraAPIURL),
				KBID:   envOr("GEEGOO_WEKNORA_KB_ID", config.DefaultWeKnoraKBID),
				APIKey: key,
			}
			return weknora.NewFromResolved(resolved)
		}
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg config.AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	return weknora.NewFromResolved(cfg.ResolvedWeKnora())
}

func optionalComposeLLM() llm.Provider {
	path := strings.TrimSpace(os.Getenv("GEEGOO_CONFIG"))
	if path == "" {
		return optionalComposeLLMFromEnv()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return optionalComposeLLMFromEnv()
	}
	var cfg config.AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return optionalComposeLLMFromEnv()
	}
	llmCfg := cfg.LLM
	if strings.TrimSpace(llmCfg.TokenKey) == "" {
		return optionalComposeLLMFromEnv()
	}
	provider, err := llm.BuildProviderFromLLMFields(
		llmCfg.Provider, llmCfg.TokenKey, llmCfg.Model,
		llmCfg.Thinking, llmCfg.ReasoningEffort, llmCfg.BaseURL,
		llmCfg.PromptCache,
	)
	if err != nil {
		return optionalComposeLLMFromEnv()
	}
	thinkingOn := llm.ResolveThinkingEnabled(llm.ProviderName(llmCfg.Provider), llmCfg.Model, llmCfg.Thinking)
	gw := llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries:  2,
		RetryWait:   time.Second,
		Temperature: llmCfg.Temperature,
		MaxTokens:   llmCfg.EffectiveMaxTokens(thinkingOn),
	})
	return llm.SynthesisProviderFromGateway(gw)
}

func optionalComposeLLMFromEnv() llm.Provider {
	token := strings.TrimSpace(os.Getenv("LLM_TOKEN_KEY"))
	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if token == "" || model == "" {
		return nil
	}
	provider, err := llm.BuildProviderFromLLMFields(
		envOr("LLM_PROVIDER", "openai"), token, model,
		nil, "", envOr("LLM_BASE_URL", ""),
		nil,
	)
	if err != nil {
		return nil
	}
	gw := llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries: 2, RetryWait: time.Second, Temperature: 0.3, MaxTokens: 2048,
	})
	return llm.SynthesisProviderFromGateway(gw)
}
