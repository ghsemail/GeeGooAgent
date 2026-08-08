package skills

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// SchedulerJobView is a dashboard-safe scheduler row (avoids importing scheduler package).
type SchedulerJobView struct {
	Name        string
	Skill       string
	Cron        string
	Enabled     bool
	LastRun     string
	LastVerdict string
}

// BuildWorkflowDetail returns dashboard-friendly workflow settings for one L5 skill.
// skillPath is the absolute path to SKILL.md from the procedural loader (preferred for reading sibling files).
func BuildWorkflowDetail(projectRoot string, spec Spec, jobs []SchedulerJobView, skillPath string) map[string]any {
	skillDir := filepath.Join("skills", spec.Name)
	detail := map[string]any{
		"name":           spec.Name,
		"description":    spec.Description,
		"skill_path":     filepath.ToSlash(filepath.Join(skillDir, "SKILL.md")),
		"workflow_path":  filepath.ToSlash(filepath.Join(skillDir, "workflow.md")),
		"manifest_path":  spec.ManifestPath,
		"template_path":  spec.TemplatePath,
		"phase_a_steps":  serializeWorkflowSteps(resolvePhaseASteps(spec)),
		"phase_b_steps":  serializeWorkflowSteps(resolvePerStockSteps(spec)),
		"scheduler_jobs": schedulerJobsForSkill(jobs, spec.Name),
	}
	root := strings.TrimSpace(projectRoot)
	if root != "" {
		readInto(detail, root, filepath.Join(skillDir, "SKILL.md"), "skill_md")
		readInto(detail, root, filepath.Join(skillDir, "workflow.md"), "workflow_md")
		if spec.ManifestPath != "" {
			readInto(detail, root, spec.ManifestPath, "manifest_yaml")
		}
		if spec.TemplatePath != "" {
			readInto(detail, root, spec.TemplatePath, "template_md")
		}
		supervisorRel := filepath.Join(skillDir, "supervisor_checks.yaml")
		readInto(detail, root, supervisorRel, "supervisor_checks_yaml")
	}
	if strings.TrimSpace(skillPath) != "" {
		enrichFromSkillDir(detail, filepath.Dir(skillPath))
	}
	return detail
}

func enrichFromSkillDir(detail map[string]any, dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	readIntoAbs(detail, filepath.Join(dir, "SKILL.md"), "skill_md")
	readIntoAbs(detail, filepath.Join(dir, "workflow.md"), "workflow_md")
	readIntoAbs(detail, filepath.Join(dir, "template.md"), "template_md")
	readIntoAbs(detail, filepath.Join(dir, "manifest.yaml"), "manifest_yaml")
	readIntoAbs(detail, filepath.Join(dir, "supervisor_checks.yaml"), "supervisor_checks_yaml")
	detail["skill_path"] = filepath.ToSlash(filepath.Join(dir, "SKILL.md"))
	detail["workflow_path"] = filepath.ToSlash(filepath.Join(dir, "workflow.md"))
	detail["template_path"] = filepath.ToSlash(filepath.Join(dir, "template.md"))
	detail["manifest_path"] = filepath.ToSlash(filepath.Join(dir, "manifest.yaml"))
}

func readInto(detail map[string]any, root, rel, key string) {
	readIntoAbs(detail, filepath.Join(root, rel), key)
}

func readIntoAbs(detail map[string]any, abs, key string) {
	if raw, err := os.ReadFile(abs); err == nil {
		detail[key] = string(raw)
	}
}

// AttachWorkflowDetails merges workflow_detail into dashboard skill rows when registered.
func AttachWorkflowDetails(items []map[string]any, projectRoot string, jobs []SchedulerJobView) {
	if len(items) == 0 {
		return
	}
	byName := map[string]Spec{}
	for _, spec := range Default().List() {
		byName[spec.Name] = spec
	}
	for i, item := range items {
		name, _ := item["name"].(string)
		name = strings.TrimSpace(name)
		spec, ok := byName[name]
		if !ok {
			continue
		}
		skillPath, _ := item["path"].(string)
		item["workflow_detail"] = BuildWorkflowDetail(projectRoot, spec, jobs, skillPath)
		items[i] = item
	}
}

func serializeWorkflowSteps(steps []workflow.Step) []map[string]any {
	if len(steps) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		row := map[string]any{
			"name": step.Name,
			"tool": step.Tool,
		}
		if len(step.Arguments) > 0 {
			row["arguments"] = step.Arguments
		}
		out = append(out, row)
	}
	return out
}

func resolvePhaseASteps(spec Spec) []workflow.Step {
	if spec.PhaseA != nil {
		if steps := spec.PhaseA(); len(steps) > 0 {
			return steps
		}
	}
	switch spec.Name {
	case "premarket_market":
		return workflow.MarketPhaseSteps(workflow.MarketCN)
	case "premarket_stock":
		return workflow.StockPhaseASteps(workflow.MarketCN)
	default:
		if spec.PhaseA != nil {
			return spec.PhaseA()
		}
		return nil
	}
}

func resolvePerStockSteps(spec Spec) []workflow.Step {
	if spec.Name == "premarket_market" {
		return nil
	}
	if spec.PerStock != nil {
		return spec.PerStock()
	}
	return nil
}

func schedulerJobsForSkill(jobs []SchedulerJobView, skill string) []map[string]any {
	out := make([]map[string]any, 0)
	for _, job := range jobs {
		if strings.TrimSpace(job.Skill) != skill {
			continue
		}
		out = append(out, map[string]any{
			"name":         job.Name,
			"skill":        job.Skill,
			"cron":         job.Cron,
			"enabled":      job.Enabled,
			"last_run":     job.LastRun,
			"last_verdict": job.LastVerdict,
		})
	}
	return out
}
