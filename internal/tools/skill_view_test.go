package tools_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestSkillViewLoadsBody(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "---\nname: demo-skill\ndescription: demo\n---\n\nstep one\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{
		HTTP:        tools.TestHTTPBackends(nil),
		WorkspaceRoot: dir,
		SkillLoader: procedural.NewLoader(dir),
	})
	res := r.Execute(tools.CallRequest{
		Name:      "skill_view",
		Arguments: map[string]any{"skill": "demo-skill"},
	}, tools.Context{})
	if res.Status != tools.StatusOK {
		t.Fatalf("status=%s summary=%s", res.Status, res.Summary)
	}
	body, _ := res.Data["body"].(string)
	if body != "step one" {
		t.Fatalf("body=%q", body)
	}
}

func TestSkillViewRejectsOtherSkillWhenLocked(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"signal_diagnose", "strategy_dev"} {
		skillDir := filepath.Join(dir, name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		text := "---\nname: " + name + "\ndescription: d\n---\n\nbody\n"
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{
		HTTP:          tools.TestHTTPBackends(nil),
		WorkspaceRoot: dir,
		SkillLoader:   procedural.NewLoader(dir),
	})
	res := r.Execute(tools.CallRequest{
		Name:      "skill_view",
		Arguments: map[string]any{"skill": "strategy_dev"},
	}, tools.Context{LockedSkill: "signal_diagnose"})
	if res.Status != tools.StatusError {
		t.Fatalf("expected error, got %s %s", res.Status, res.Summary)
	}
	ok := r.Execute(tools.CallRequest{
		Name:      "skill_view",
		Arguments: map[string]any{"skill": "signal_diagnose"},
	}, tools.Context{LockedSkill: "signal_diagnose"})
	if ok.Status != tools.StatusOK {
		t.Fatalf("locked skill should load: %s", ok.Summary)
	}
}
