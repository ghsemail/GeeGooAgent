package procedural_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
)

func TestFilterSchemasForMatchedSkillsBacktestRun(t *testing.T) {
	schemas := []llm.ToolSchema{
		{Name: "run_strategy_backtest"},
		{Name: "generate_dca_strategy"},
		{Name: "loopback_strategy"},
	}
	out := procedural.FilterSchemasForMatchedSkills(
		schemas,
		[]string{"strategy-backtest-run"},
		"帮我回测一下",
		nil,
	)
	if len(out) != 1 || out[0].Name != "run_strategy_backtest" {
		t.Fatalf("out=%v", out)
	}
}

func TestFilterSchemasForMatchedSkillsNoSkillMatch(t *testing.T) {
	schemas := []llm.ToolSchema{{Name: "generate_dca_strategy"}}
	out := procedural.FilterSchemasForMatchedSkills(schemas, nil, "帮我回测一下", nil)
	if len(out) != 1 {
		t.Fatalf("out=%v", out)
	}
}
