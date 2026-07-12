package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApplyProgressLLMPlanCreatesThinking(t *testing.T) {
	m := NewModel(config.DisplayConfig{DetailsMode: config.ModeCollapsed}, nil, nil)
	m.ApplyProgress("llm_plan", map[string]any{"reasoning": "find ticker first"})
	if len(m.blocks) != 1 || m.blocks[0].Kind != KindThinking {
		t.Fatalf("blocks=%+v", m.blocks)
	}
	if !m.blocks[0].Live || !m.blocks[0].IsExpanded(m.display) {
		t.Fatal("live thinking should expand")
	}
	m.finalizeLiveSections()
	if m.blocks[0].Live || m.blocks[0].IsExpanded(m.display) {
		t.Fatal("after finalize, collapsed mode should hide body")
	}
}

func TestShouldUseTUIForceCLI(t *testing.T) {
	if ShouldUseTUI(&config.AppConfig{}, false, true) {
		t.Fatal("force CLI")
	}
}
