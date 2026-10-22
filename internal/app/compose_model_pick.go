package app

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SignalDiagnoseComposeProvider returns the LLM for signal_diagnose diag_compose only.
func (a *App) SignalDiagnoseComposeProvider() llm.Provider {
	if a == nil {
		return nil
	}
	return a.cachedSignalDiagnoseComposeProvider()
}

// ComposeBenchmarkProvider builds a classify-policy provider for one catalog row (benchmarks only).
func (a *App) ComposeBenchmarkProvider(doc admin.ConfiguredModel) (llm.Provider, error) {
	return a.composeProviderFromCatalogDoc(doc)
}

func (a *App) composeProviderFromCatalogDoc(doc admin.ConfiguredModel) (llm.Provider, error) {
	providerName := a.Config.LLM.Provider
	tokenKey := a.Config.LLM.TokenKey
	model := a.Config.LLM.Model
	baseURL := a.Config.LLM.BaseURL
	a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
	thinkingOff := false
	provider, err := llm.BuildProviderFromLLMFields(
		providerName, tokenKey, model, &thinkingOff, "", baseURL, nil,
	)
	if err != nil {
		return nil, err
	}
	gw := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gw.SetPolicy(a.buildModelPolicy(false, a.Config.LLM.EffectiveMaxTokens(false), a.Config.LLM.Temperature))
	return llm.ClassifyProviderFromGateway(gw), nil
}

// ListCatalogComposeCandidates returns non-forced-reasoning catalog rows for benchmarking.
func (a *App) ListCatalogComposeCandidates(ctx context.Context) ([]admin.ConfiguredModel, error) {
	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		return nil, fmt.Errorf("no catalog targets")
	}
	models, _, err := admin.ListModelsFromTargets(ctx, targets)
	if err != nil {
		return nil, err
	}
	out := make([]admin.ConfiguredModel, 0, len(models))
	for _, m := range models {
		if isComposeCatalogExcluded(m) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}
