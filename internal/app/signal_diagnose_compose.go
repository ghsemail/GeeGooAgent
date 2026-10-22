package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SignalDiagnoseComposeProvider returns the LLM for signal_diagnose diag_compose only.
// Default: chat gateway with TaskClassify policy (pre-PR#69 compose path; not ops synthesis gateway).
// Override: SIGNAL_DIAGNOSE_COMPOSE_MODEL=<ops catalog model id or name>.
func (a *App) SignalDiagnoseComposeProvider() llm.Provider {
	if a == nil {
		return nil
	}
	return a.cachedSignalDiagnoseComposeProvider()
}

func (a *App) signalDiagnoseComposeFromEnv() llm.Provider {
	id := strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_MODEL"))
	if id == "" || a.Config == nil {
		return nil
	}
	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "警告: SIGNAL_DIAGNOSE_COMPOSE_MODEL 未配置 catalog targets\n")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, _, err := admin.QueryModelFromTargets(ctx, targets, id, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: SIGNAL_DIAGNOSE_COMPOSE_MODEL=%q: %v\n", id, err)
		return nil
	}
	providerName := a.Config.LLM.Provider
	tokenKey := a.Config.LLM.TokenKey
	model := a.Config.LLM.Model
	baseURL := a.Config.LLM.BaseURL
	a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
	thinkingOff := false
	provider, err := llm.BuildProviderFromLLMFields(
		providerName,
		tokenKey,
		model,
		&thinkingOff,
		"",
		baseURL,
		nil,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: compose 模型 %q 构建失败: %v\n", id, err)
		return nil
	}
	gw := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	temp := a.Config.LLM.Temperature
	maxTok := a.Config.LLM.EffectiveMaxTokens(false)
	gw.SetPolicy(a.buildModelPolicy(false, maxTok, temp))
	return llm.ClassifyProviderFromGateway(gw)
}
