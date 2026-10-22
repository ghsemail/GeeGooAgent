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

const (
	ComposeSourceEnv             = "env"
	ComposeSourceCatalogPref     = "catalog_pref"
	ComposeSourceGatewayFallback = "gateway_fallback"
	ComposeSourceOpsFallback     = "ops_classify_fallback"
)

// ComposeLLMBundle is the resolved signal_diagnose compose model and provenance.
type ComposeLLMBundle struct {
	Provider llm.Provider
	Source   string
	Detail   string
}

var (
	composeBundleMu          sync.Mutex
	cachedBundle             *ComposeLLMBundle
	lastCatalogComposeDetail string
)

// ResolveSignalDiagnoseComposeBundle picks compose LLM (re-tries catalog when last fallback failed).
func (a *App) ResolveSignalDiagnoseComposeBundle() ComposeLLMBundle {
	composeBundleMu.Lock()
	defer composeBundleMu.Unlock()
	if cachedBundle != nil && (cachedBundle.Source == ComposeSourceCatalogPref || cachedBundle.Source == ComposeSourceEnv) {
		return *cachedBundle
	}
	b := a.buildSignalDiagnoseComposeBundle()
	if b.Source == ComposeSourceCatalogPref || b.Source == ComposeSourceEnv {
		cachedBundle = &b
	}
	return b
}

func (a *App) invalidateComposeProviderCache() {
	composeBundleMu.Lock()
	cachedBundle = nil
	composeBundleMu.Unlock()
}

func (a *App) cachedSignalDiagnoseComposeProvider() llm.Provider {
	return a.ResolveSignalDiagnoseComposeBundle().Provider
}

func (a *App) buildSignalDiagnoseComposeBundle() ComposeLLMBundle {
	if p, src, detail := a.signalDiagnoseComposeFromEnvWithMeta(); p != nil {
		return ComposeLLMBundle{Provider: p, Source: src, Detail: detail}
	}
	if p, src, detail := a.signalDiagnoseComposeFromCatalogPrefsWithMeta(); p != nil {
		return ComposeLLMBundle{Provider: p, Source: src, Detail: detail}
	}
	fallbackDetail := func(model string) string {
		d := "chat_gateway_classify:" + strings.TrimSpace(model)
		if lastCatalogComposeDetail != "" {
			d += ";catalog:" + lastCatalogComposeDetail
		}
		return d
	}
	if a.Agent != nil && a.Agent.Gateway != nil {
		return ComposeLLMBundle{
			Provider: llm.ClassifyProviderFromGateway(a.Agent.Gateway),
			Source:   ComposeSourceGatewayFallback,
			Detail:   fallbackDetail(a.Agent.Gateway.Model()),
		}
	}
	if a.Gateway != nil {
		return ComposeLLMBundle{
			Provider: llm.ClassifyProviderFromGateway(a.Gateway),
			Source:   ComposeSourceGatewayFallback,
			Detail:   fallbackDetail(a.Gateway.Model()),
		}
	}
	if p := a.classifyOpsFallbackProvider(); p != nil {
		return ComposeLLMBundle{Provider: p, Source: ComposeSourceOpsFallback, Detail: "ops_classify_fallback"}
	}
	return ComposeLLMBundle{Source: ComposeSourceGatewayFallback, Detail: "no_llm_configured"}
}

func (a *App) signalDiagnoseComposeFromEnvWithMeta() (llm.Provider, string, string) {
	id := strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_MODEL"))
	if id == "" {
		return nil, "", ""
	}
	p := a.signalDiagnoseComposeFromEnv()
	if p == nil {
		return nil, ComposeSourceEnv, "env_compose_model_unresolved:" + id
	}
	return p, ComposeSourceEnv, id
}

func (a *App) signalDiagnoseComposeFromCatalogPrefsWithMeta() (llm.Provider, string, string) {
	if a == nil || a.Config == nil {
		return nil, "", ""
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_USE_CHAT_MODEL")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_USE_CHAT_MODEL")), "true") {
		return nil, "", ""
	}
	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		lastCatalogComposeDetail = "no_catalog_targets"
		return nil, "", ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	models, baseURL, err := admin.ListModelsFromTargets(ctx, targets)
	if err != nil {
		lastCatalogComposeDetail = "list_failed:" + err.Error()
		fmt.Fprintf(os.Stderr, "signal_diagnose compose: catalog list failed: %v\n", err)
		return nil, "", ""
	}
	if len(models) == 0 {
		lastCatalogComposeDetail = "catalog_empty:" + baseURL
		fmt.Fprintf(os.Stderr, "signal_diagnose compose: catalog empty (base=%s)\n", baseURL)
		return nil, "", ""
	}
	doc, pickDetail := pickCatalogComposeModelWithDetail(models, signalDiagnoseComposeCatalogPrefs)
	if doc.Name == "" || strings.TrimSpace(doc.Token) == "" {
		lastCatalogComposeDetail = pickDetail
		fmt.Fprintf(os.Stderr, "signal_diagnose compose: catalog no pref match (%s)\n", pickDetail)
		return nil, "", ""
	}
	lastCatalogComposeDetail = ""
	p, err := a.composeProviderFromCatalogDoc(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signal_diagnose compose: build provider %q: %v\n", doc.Name, err)
		return nil, "", ""
	}
	fmt.Fprintf(os.Stderr, "signal_diagnose compose: catalog_pref %q\n", doc.Name)
	return p, ComposeSourceCatalogPref, doc.Name
}

func pickCatalogComposeModelWithDetail(models []admin.ConfiguredModel, prefs []string) (admin.ConfiguredModel, string) {
	for _, pref := range prefs {
		for _, m := range models {
			if isComposeCatalogExcluded(m) {
				continue
			}
			if catalogNameEquals(m, pref) {
				return m, "matched:" + pref
			}
		}
	}
	var excluded, noToken int
	for _, m := range models {
		if strings.TrimSpace(m.Token) == "" {
			noToken++
			continue
		}
		if isComposeCatalogExcluded(m) {
			excluded++
		}
	}
	return admin.ConfiguredModel{}, fmt.Sprintf("no_match prefs=%v rows=%d excluded=%d no_token=%d", prefs, len(models), excluded, noToken)
}
