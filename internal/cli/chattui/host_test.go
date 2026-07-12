package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApplyVerboseToDisplay(t *testing.T) {
	d := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	ApplyVerboseToDisplay(&d, true)
	if d.EffectiveMode("thinking") != config.ModeExpanded {
		t.Fatalf("thinking=%s", d.EffectiveMode("thinking"))
	}
	ApplyVerboseToDisplay(&d, false)
	if d.EffectiveMode("thinking") != config.ModeCollapsed {
		t.Fatalf("thinking=%s", d.EffectiveMode("thinking"))
	}
}
