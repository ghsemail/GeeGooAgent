package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildWorkflowDetailPreMarketMarket(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("pre_market_market")
	if !ok {
		t.Fatal("pre_market_market not registered")
	}
	jobs := []SchedulerJobView{
		{Name: "pre_market_market_cn", Skill: "pre_market_market", Cron: "0 8 * * 1-5", Enabled: true},
	}
	detail := BuildWorkflowDetail(root, spec, jobs, filepath.Join(root, "skills", "pre_market_market", "SKILL.md"))
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
	}
	if detail["skill_md"] == nil {
		t.Fatal("expected skill_md")
	}
}

func TestBuildWorkflowDetailPreMarketStock(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("pre_market_stock")
	if !ok {
		t.Fatal("pre_market_stock not registered")
	}
	detail := BuildWorkflowDetail(root, spec, []SchedulerJobView{
		{Name: "pre_market_stock_cn", Skill: "pre_market_stock", Cron: "10 8 * * 1-5", Enabled: true},
	}, filepath.Join(root, "skills", "pre_market_stock", "SKILL.md"))
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
	}
	if len(detail["phase_b_steps"].([]map[string]any)) < 5 {
		t.Fatal("expected pre_market_stock per-stock steps")
	}
}

func TestBuildWorkflowDetailPostMarket(t *testing.T) {
	root := findRepoRoot(t)
	spec, ok := Default().Get("post_market")
	if !ok {
		t.Fatal("post_market not registered")
	}
	detail := BuildWorkflowDetail(root, spec, []SchedulerJobView{
		{Name: "post_market_weekday", Skill: "post_market", Cron: "0 17 * * 1-5", Enabled: true},
	}, filepath.Join(root, "skills", "post_market", "SKILL.md"))
	if detail["manifest_yaml"] == nil {
		t.Fatal("expected manifest_yaml")
	}
	if len(detail["phase_b_steps"].([]map[string]any)) < 5 {
		t.Fatal("expected post_market per-stock steps")
	}
}

func TestAttachWorkflowDetails(t *testing.T) {
	items := []map[string]any{
		{"name": "pre_market_market", "kind": "workflow"},
		{"name": "bot-manager", "kind": "playbook"},
	}
	AttachWorkflowDetails(items, findRepoRoot(t), []SchedulerJobView{
		{Name: "pre_market_market_cn", Skill: "pre_market_market", Cron: "0 8 * * 1-5", Enabled: true},
	})
	if items[0]["workflow_detail"] == nil {
		t.Fatal("expected workflow_detail on pre_market_market")
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
		if _, err := os.Stat(filepath.Join(dir, "skills", "pre_market_market", "manifest.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root with skills/pre_market/manifest.yaml")
	return ""
}
