package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

type llmSource string

const (
	llmSourceCatalog llmSource = "catalog"
	llmSourceOps     llmSource = "ops"
	llmSourceLocal   llmSource = "local"
)

type resolvedLLMFields struct {
	provider, tokenKey, model, baseURL string
	source                             llmSource
	catalogSrc                         string
}

func (a *App) adminQueryTargets() []admin.QueryTarget {
	if a == nil || a.Config == nil {
		return nil
	}
	targets := make([]admin.QueryTarget, 0, len(a.Config.AdminModelQueryTargets()))
	for _, t := range a.Config.AdminModelQueryTargets() {
		targets = append(targets, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
	}
	return targets
}

// resolveLLMFromConfig picks the chat LLM from one explicit source:
//  1. catalog_model_id → catalog-api row (by ObjectId or name alias)
//  2. use_ops_model (default true when unset) → ops configured primary
//  3. otherwise → provider/model/token_key in cfg (local fallback)
func (a *App) resolveLLMFromConfig(cfg config.LLMConfig) (*resolvedLLMFields, error) {
	if a == nil || a.Config == nil {
		return nil, fmt.Errorf("app not configured")
	}
	providerName := cfg.Provider
	tokenKey := cfg.TokenKey
	model := cfg.Model
	baseURL := strings.TrimSpace(cfg.BaseURL)
	out := &resolvedLLMFields{
		provider: providerName,
		tokenKey: tokenKey,
		model:    model,
		baseURL:  baseURL,
		source:   llmSourceLocal,
	}

	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		if strings.TrimSpace(cfg.CatalogModelID) != "" || cfg.OpsModelEnabled() {
			return nil, fmt.Errorf("no catalog query targets configured")
		}
		return out, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if id := strings.TrimSpace(cfg.CatalogModelID); id != "" {
		doc, src, err := admin.QueryModelFromTargets(ctx, targets, id, false)
		if err != nil {
			return nil, fmt.Errorf("catalog model %q: %w", id, err)
		}
		a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
		out.provider = providerName
		out.tokenKey = tokenKey
		out.model = model
		out.baseURL = baseURL
		out.source = llmSourceCatalog
		out.catalogSrc = src
		return out, nil
	}

	if cfg.OpsModelEnabled() {
		doc, src, err := admin.QueryConfiguredFromTargets(ctx, targets...)
		if err != nil {
			return nil, fmt.Errorf("ops configured model: %w", err)
		}
		a.applyCatalogModelDoc(&doc, &providerName, &tokenKey, &model, &baseURL)
		out.provider = providerName
		out.tokenKey = tokenKey
		out.model = model
		out.baseURL = baseURL
		out.source = llmSourceOps
		out.catalogSrc = src
		return out, nil
	}

	return out, nil
}
