package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApplyProgressLLMPlanCreatesThinking(t *testing.T) {
	s := &LiveSlot{Status: "ready", Focus: -1}
	s.ApplyProgress("llm_plan", map[string]any{"reasoning": "find ticker first"})
	if len(s.Blocks) != 1 || s.Blocks[0].Kind != KindThinking {
		t.Fatalf("blocks=%+v", s.Blocks)
	}
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	if !s.Blocks[0].Live || s.Blocks[0].IsExpanded(cfg) {
		t.Fatal("live thinking should use compact preview in collapsed mode")
	}
	if !s.Blocks[0].ShowThinkingPreview(cfg) {
		t.Fatal("live thinking should show thinking preview")
	}
	s.finalizeLiveSections()
	if s.Blocks[0].Live || s.Blocks[0].IsExpanded(cfg) {
		t.Fatal("after finalize, collapsed mode should hide body")
	}
}

func TestShouldUseTUIForceCLI(t *testing.T) {
	if ShouldUseTUI(&config.AppConfig{}, false, true) {
		t.Fatal("force CLI")
	}
}
