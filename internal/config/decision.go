package config

import "strings"

// DecisionConfig holds ops catalog Jev model metadata (runtime calls use GeeGooData Decision API + data service token).
type DecisionConfig struct {
	TokenKey string `json:"token_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url,omitempty"`
}

const (
	DefaultDecisionModel  = "jev-1.13.0"
	DefaultDecisionBaseURL  = "https://api.typesafe.ai/v1"
	DefaultDecisionProvider = "typesafe"
)

// Decision hook modes (in-loop Jev consult; default off).
const (
	DecisionHookModeOff     = "off"
	DecisionHookModeShadow  = "shadow"
	DecisionHookModeReplace = "replace"
	DecisionHookModePaper   = "paper"
)

// ResolvedDecision is EffectiveDecision output.
type ResolvedDecision struct {
	Provider   string
	TokenKey   string
	Model      string
	BaseURL    string
	Configured bool
}

// EffectiveDecisionHookMode returns in-loop Jev consult mode (default off).
func (c *AppConfig) EffectiveDecisionHookMode() string {
	if c == nil {
		return DecisionHookModeOff
	}
	switch strings.ToLower(strings.TrimSpace(c.DecisionHookMode)) {
	case DecisionHookModeShadow, DecisionHookModeReplace, DecisionHookModePaper:
		return strings.ToLower(strings.TrimSpace(c.DecisionHookMode))
	default:
		return DecisionHookModeOff
	}
}

// ResolvedDecision returns decision settings from config.
func (c *AppConfig) ResolvedDecision() ResolvedDecision {
	var cfg DecisionConfig
	if c != nil {
		cfg = c.Decision
	}
	key := strings.TrimSpace(cfg.TokenKey)
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultDecisionModel
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = DefaultDecisionBaseURL
	}
	return ResolvedDecision{
		Provider:   DefaultDecisionProvider,
		TokenKey:   key,
		Model:      model,
		BaseURL:    base,
		Configured: key != "" && strings.TrimSpace(cfg.Model) != "",
	}
}
