package chatcmd

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApplyDetailsGlobal(t *testing.T) {
	d := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	res := ApplyDetails(&d, []string{"expanded"})
	if !res.OK || !res.Persist || res.Display.DetailsMode != config.ModeExpanded {
		t.Fatalf("%+v", res)
	}
}

func TestApplyDetailsCycle(t *testing.T) {
	d := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	res := ApplyDetails(&d, []string{"cycle"})
	if res.Display.DetailsMode != config.ModeExpanded {
		t.Fatalf("got %s", res.Display.DetailsMode)
	}
}

func TestApplyDetailsSection(t *testing.T) {
	d := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	res := ApplyDetails(&d, []string{"thinking", "expanded"})
	if !res.OK || res.Display.EffectiveMode("thinking") != config.ModeExpanded {
		t.Fatalf("%+v mode=%s", res, res.Display.EffectiveMode("thinking"))
	}
}

func TestApplyDetailsLast(t *testing.T) {
	d := config.DisplayConfig{}
	res := ApplyDetails(&d, []string{"last"})
	if !res.OK || !res.ShowLast || res.Persist {
		t.Fatalf("%+v", res)
	}
}

func TestApplyDetailsStatus(t *testing.T) {
	d := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	res := ApplyDetails(&d, nil)
	if !res.OK || res.Persist || !strings.Contains(res.Message, "details_mode=collapsed") {
		t.Fatalf("%+v", res)
	}
}
