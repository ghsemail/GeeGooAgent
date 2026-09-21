package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const opsDecisionRefreshEvery = 60 * time.Second

func catalogDecisionConfig(doc admin.ConfiguredModel) config.DecisionConfig {
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		name = strings.TrimSpace(doc.DisplayName)
	}
	base := strings.TrimRight(strings.TrimSpace(doc.BaseURL), "/")
	if base == "" {
		base = config.DefaultDecisionBaseURL
	}
	return config.DecisionConfig{
		TokenKey: strings.TrimSpace(doc.Token),
		Model:    name,
		BaseURL:  base,
	}
}

func catalogModelIsDecision(doc admin.ConfiguredModel) bool {
	switch strings.ToLower(strings.TrimSpace(doc.Kind)) {
	case "decision", "systemone", "system_one", "jev", "typesafe":
		return true
	}
	hay := strings.ToLower(doc.Name + " " + doc.DisplayName + " " + doc.BaseURL)
	return strings.Contains(hay, "typesafe") || strings.Contains(hay, "jev")
}

func applyCatalogDecision(cfg *config.AppConfig, doc admin.ConfiguredModel) bool {
	if cfg == nil || !catalogModelIsDecision(doc) {
		return false
	}
	next := catalogDecisionConfig(doc)
	if strings.TrimSpace(next.TokenKey) == "" || strings.TrimSpace(next.Model) == "" {
		return false
	}
	cfg.Decision = next
	return true
}

func (a *App) RefreshOpsDecision(force bool) {
	if a == nil || a.Config == nil {
		return
	}
	a.decisionMu.Lock()
	defer a.decisionMu.Unlock()
	if !force && !a.decisionRefreshedAt.IsZero() && time.Since(a.decisionRefreshedAt) < opsDecisionRefreshEvery {
		return
	}
	targets := a.catalogQueryTargets()
	if len(targets) == 0 {
		a.decisionSource = "unset"
		a.decisionRefreshedAt = time.Now()
		return
	}
	view, err := admin.QueryModelRuntimeConfigFromTargets(context.Background(), targets)
	if err != nil {
		a.decisionSource = "unset"
		a.decisionRefreshedAt = time.Now()
		return
	}
	if view.DecisionModel == nil || strings.TrimSpace(view.DecisionModelID) == "" {
		a.decisionSource = "unset"
		a.decisionRefreshedAt = time.Now()
		return
	}
	if !catalogModelIsDecision(*view.DecisionModel) {
		a.decisionSource = "unset"
		fmt.Fprintf(os.Stderr, "警告: 运营 decision_model_id 不是决策（Jev）类型，请检查模型管理\n")
		a.decisionRefreshedAt = time.Now()
		return
	}
	if !applyCatalogDecision(a.Config, *view.DecisionModel) {
		a.decisionSource = "unset"
		a.decisionRefreshedAt = time.Now()
		return
	}
	a.decisionSource = "ops"
	a.decisionCatalogID = strings.TrimSpace(view.DecisionModelID)
	a.decisionRefreshedAt = time.Now()
	res := a.Config.ResolvedDecision()
	fmt.Fprintf(os.Stderr, "decision: model=%s base_url=%s from ops catalog\n", res.Model, res.BaseURL)
}

func (a *App) DecisionSource() string {
	if a == nil {
		return ""
	}
	a.decisionMu.Lock()
	defer a.decisionMu.Unlock()
	if strings.TrimSpace(a.decisionSource) == "" {
		return "unset"
	}
	return a.decisionSource
}

func (a *App) DecisionCatalogID() string {
	if a == nil {
		return ""
	}
	a.decisionMu.Lock()
	defer a.decisionMu.Unlock()
	return a.decisionCatalogID
}
