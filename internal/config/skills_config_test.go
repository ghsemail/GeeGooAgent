package config_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestEffectiveSkillInjectionDefaultsCatalog(t *testing.T) {
	var c config.AppConfig
	if c.EffectiveSkillInjection() != "catalog" {
		t.Fatalf("default injection mode")
	}
}

func TestEffectiveChatOpenFlowToolDefaultFalse(t *testing.T) {
	var c config.AppConfig
	if c.EffectiveChatOpenFlowTool() {
		t.Fatal("open flow tool should default false")
	}
}
