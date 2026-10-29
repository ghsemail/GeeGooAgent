package config_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestEffectiveDecisionHookMode(t *testing.T) {
	var empty config.AppConfig
	if empty.EffectiveDecisionHookMode() != config.DecisionHookModeOff {
		t.Fatal("default off")
	}
	c := &config.AppConfig{DecisionHookMode: "shadow"}
	if c.EffectiveDecisionHookMode() != config.DecisionHookModeShadow {
		t.Fatalf("got %q", c.EffectiveDecisionHookMode())
	}
}
