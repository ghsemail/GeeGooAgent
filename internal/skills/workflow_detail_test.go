package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWorkflowDetailPreMarket(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("premarket_market")
	if !ok {
		t.Fatal("premarket_market not registered")
	}
	jobs := []SchedulerJobView{
		{Name: "premarket_market_cn", Skill: "premarket_market", Cron: "0 8 * * 1-5", Enabled: true},
	}
	detail := BuildWorkflowDetail(root, spec, jobs, filepath.Join(root, "skills", "premarket_market", "SKILL.md"))
	if detail["skill_md"] == nil {
		t.Fatal("expected skill_md")
	}
	if detail["template_md"] == nil {
		t.Fatal("expected template_md for premarket_market")
	}
}

func TestBuildWorkflowDetailPreMarketStock(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("premarket_stock")
	if !ok {
		t.Fatal("premarket_stock not registered")
	}
	detail := BuildWorkflowDetail(root, spec, []SchedulerJobView{
		{Name: "premarket_stock_cn", Skill: "premarket_stock", Cron: "10 8 * * 1-5", Enabled: true},
	}, filepath.Join(root, "skills", "premarket_stock", "SKILL.md"))
	if detail["skill_md"] == nil {
		t.Fatal("expected skill_md")
	}
	if detail["template_md"] == nil {
		t.Fatal("expected template_md for premarket_stock")
	}
	if len(detail["phase_b_steps"].([]map[string]any)) < 5 {
		t.Fatal("expected premarket_stock per-stock steps")
	}
}

func TestBuildWorkflowDetailPostMarket(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("postmarket_stock")
	if !ok {
		t.Fatal("postmarket_stock not registered")
	}
	detail := BuildWorkflowDetail(root, spec, []SchedulerJobView{
		{Name: "postmarket_stock_weekday", Skill: "postmarket_stock", Cron: "0 17 * * 1-5", Enabled: true},
	}, filepath.Join(root, "skills", "postmarket_stock", "SKILL.md"))
	if detail["skill_md"] == nil {
		t.Fatal("expected skill_md")
	}
	if len(detail["phase_b_steps"].([]map[string]any)) < 5 {
		t.Fatal("expected postmarket_stock per-stock steps")
	}
}

func TestBuildWorkflowDetailStrategyArchiveTemplate(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get(SkillGenerateStrategyArchive)
	if !ok {
		t.Fatal("missing generate_strategy_archive")
	}
	if spec.TemplatePath != "skills/generate_strategy_archive/template.md" {
		t.Fatalf("TemplatePath=%q", spec.TemplatePath)
	}
	detail := BuildWorkflowDetail(root, spec, nil, filepath.Join(root, "skills", "generate_strategy_archive", "SKILL.md"))
	raw, _ := detail["template_md"].(string)
	if !strings.Contains(raw, "doc_type: strategy_agent_archive") || !strings.Contains(raw, "{{strategy_name}}") {
		t.Fatalf("template_md missing archive output placeholders: %s", raw)
	}
	if !strings.Contains(raw, "> 1–2 句") {
		t.Fatalf("template_md should show writing hints: %s", raw)
	}
}

func TestBuildWorkflowDetailHidesLegacyCognitionTriggers(t *testing.T) {
	spec, ok := Default().Get(SkillGenerateStrategyCognition)
	if !ok {
		t.Fatal("missing generate_strategy_cognition")
	}
	detail := BuildWorkflowDetail(findRepoRoot(t), spec, nil, "")
	raw, _ := detail["chat_triggers"].([]string)
	for _, tgr := range raw {
		if strings.Contains(tgr, "认知") {
			t.Fatalf("legacy trigger visible: %v", raw)
		}
	}
}

func TestAttachWorkflowDetails(t *testing.T) {
	items := []map[string]any{
		{"name": "premarket_market", "kind": "workflow"},
		{"name": "bot-manager", "kind": "playbook"},
	}
	AttachWorkflowDetails(items, findRepoRoot(t), []SchedulerJobView{
		{Name: "premarket_market_cn", Skill: "premarket_market", Cron: "0 8 * * 1-5", Enabled: true},
	})
	if items[0]["workflow_detail"] == nil {
		t.Fatal("expected workflow_detail on premarket_market")
	}
	if items[1]["workflow_detail"] != nil {
		t.Fatal("playbook should not get workflow_detail")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "skills", "premarket_market", "SKILL.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root with skills/premarket_market/SKILL.md")
	return ""
}
