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

// Preferred signal_diagnose compose models (fast chat / non-forced-reasoning), catalog name substrings.
// Benchmark: scripts/eval/compose_model_benchmark — PR#69 前 9–11s 路径为 TaskClassify + 聊天非 M3 模型。
var signalDiagnoseComposeCatalogPrefs = []string{
	"deepseek-chat",
	"deepseek-v3",
	"gpt-4o-mini",
	"gpt-4.1-mini",
	"qwen-plus",
	"qwen-turbo",
	"MiniMax-M2.5-highspeed",
	"MiniMax-M2.5",
	"MiniMax-M2.1-highspeed",
	"MiniMax-M2.1",
}

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
	// Skip if picked model is the same slow reasoning chat primary (MiniMax-M3) unless disabled params apply.
	if strings.EqualFold(strings.TrimSpace(doc.Name), strings.TrimSpace(a.EffectiveLLMModel())) &&
		llm.ModelCatalogLikelyForcedReasoning(doc.Name) {
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

func pickCatalogComposeModel(models []admin.ConfiguredModel, prefs []string) admin.ConfiguredModel {
	byName := func(name string) (admin.ConfiguredModel, bool) {
		want := strings.ToLower(strings.TrimSpace(name))
		for _, m := range models {
			if strings.TrimSpace(m.Token) == "" {
				continue
			}
			hay := strings.ToLower(strings.TrimSpace(m.Name + " " + m.DisplayName))
			if llm.ModelCatalogLikelyForcedReasoning(hay) {
				continue
			}
			if strings.Contains(hay, want) {
				return m, true
			}
		}
		return admin.ConfiguredModel{}, false
	}
	for _, pref := range prefs {
		if m, ok := byName(pref); ok {
			return m
		}
	}
	for _, m := range models {
		if strings.TrimSpace(m.Token) == "" {
			continue
		}
		hay := strings.ToLower(m.Name + " " + m.DisplayName)
		if llm.ModelCatalogLikelyForcedReasoning(hay) {
			continue
		}
		if strings.Contains(hay, "minimax-m3") {
			continue
		}
		return m
	}
	return admin.ConfiguredModel{}
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
		if strings.TrimSpace(m.Token) == "" {
			continue
		}
		hay := strings.ToLower(m.Name + " " + m.DisplayName)
		if llm.ModelCatalogLikelyForcedReasoning(hay) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}
