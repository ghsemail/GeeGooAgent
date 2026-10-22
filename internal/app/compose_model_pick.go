package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var (
	composeProviderMu sync.Mutex
	cachedComposeLLM  llm.Provider
)

func (a *App) cachedSignalDiagnoseComposeProvider() llm.Provider {
	composeProviderMu.Lock()
	defer composeProviderMu.Unlock()
	if cachedComposeLLM != nil {
		return cachedComposeLLM
	}
	cachedComposeLLM = a.buildSignalDiagnoseComposeProvider()
	return cachedComposeLLM
}

func (a *App) invalidateComposeProviderCache() {
	composeProviderMu.Lock()
	cachedComposeLLM = nil
	composeProviderMu.Unlock()
}

func (a *App) buildSignalDiagnoseComposeProvider() llm.Provider {
	if p := a.signalDiagnoseComposeFromEnv(); p != nil {
		return p
	}
	if p := a.signalDiagnoseComposeFromCatalogPrefs(); p != nil {
		return p
	}
	if a.Agent != nil && a.Agent.Gateway != nil {
		return llm.ClassifyProviderFromGateway(a.Agent.Gateway)
	}
	if a.Gateway != nil {
		return llm.ClassifyProviderFromGateway(a.Gateway)
	}
	return a.classifyOpsFallbackProvider()
}

func (a *App) signalDiagnoseComposeFromCatalogPrefs() llm.Provider {
	if a == nil || a.Config == nil {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_USE_CHAT_MODEL")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_USE_CHAT_MODEL")), "true") {
		return nil
	}
	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	models, _, err := admin.ListModelsFromTargets(ctx, targets)
	if err != nil || len(models) == 0 {
		return nil
	}
	doc := pickCatalogComposeModel(models, signalDiagnoseComposeCatalogPrefs)
	if doc.Name == "" && doc.Token == "" {
		return nil
	}
	p, err := a.composeProviderFromCatalogDoc(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: compose catalog 模型 %q: %v\n", doc.Name, err)
		return nil
	}
	fmt.Fprintf(os.Stderr, "signal_diagnose compose: 使用 catalog 模型 %q（非聊天主模型）\n", doc.Name)
	return p
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
