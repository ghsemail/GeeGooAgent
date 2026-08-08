package skills

import (
	"os"
	"path/filepath"
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
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
	}
	if detail["skill_md"] == nil {
		t.Fatal("expected skill_md")
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
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
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
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
	}
	if len(detail["phase_b_steps"].([]map[string]any)) < 5 {
		t.Fatal("expected postmarket_stock per-stock steps")
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
		if _, err := os.Stat(filepath.Join(dir, "skills", "premarket_market", "manifest.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root with skills/premarket_market/manifest.yaml")
	return ""
}
