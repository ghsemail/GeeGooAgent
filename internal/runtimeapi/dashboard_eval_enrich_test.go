package runtimeapi

import (
	"encoding/json"
	"testing"
)

func TestPersistEvalCaseOptionsKeepsSessionCleanup(t *testing.T) {
	in := map[string]any{
		"category": "turn_plan",
		"session_cleanup": "never",
		"dual_model_eval": true,
		"random_stock_enabled": false,
		"message": "hello",
		"expect_domain": "backtest_run",
	}
	out := persistEvalCaseOptions(in)
	if out["session_cleanup"] != "before_run" {
		t.Fatalf("session_cleanup=%v want before_run", out["session_cleanup"])
	}
	if out["dual_model_eval"] != true {
		t.Fatalf("dual_model_eval=%v", out["dual_model_eval"])
	}
	raw, _ := json.Marshal(out)
	if !json.Valid(raw) {
		t.Fatal("invalid json")
	}
}

func TestEnrichEvalCaseRowTurnPlanGroup(t *testing.T) {
	row := map[string]any{
		"options": map[string]any{
			"category":         "turn_plan",
			"turn_plan_group":    "stock_analysis",
			"turn_id":            "subagent_multi_stock_price",
			"message":            "请帮我分析下腾讯和阿里巴巴最近的股价",
		},
	}
	enrichEvalCaseRow(row)
	if row["turn_plan_group"] != "stock_analysis" {
		t.Fatalf("turn_plan_group=%v", row["turn_plan_group"])
	}
	if row["turn_plan_group_title"] != "股票分析" {
		t.Fatalf("turn_plan_group_title=%v", row["turn_plan_group_title"])
	}
}

func TestPersistEvalCaseOptionsKeepsRuntimePrefsAfterScriptShape(t *testing.T) {
	in := map[string]any{
		"category":         "turn_plan",
		"message":          "帮我用这些策略回测一下",
		"expect_domain":    "backtest_run",
		"expect_mode":      "execute",
		"session_cleanup":  "never",
		"dual_model_eval":  false,
		"dialogue": []any{
			map[string]any{"role": "user", "text": "帮我用这些策略回测一下", "judge": true},
		},
	}
	out := persistEvalCaseOptions(in)
	if out["session_cleanup"] != "before_run" {
		t.Fatalf("session_cleanup=%v want before_run", out["session_cleanup"])
	}
}
