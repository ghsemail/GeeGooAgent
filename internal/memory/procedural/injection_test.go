package procedural_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
)

func TestChatCatalogExcludesScheduledRunSkills(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "signal_diagnose", "Chat signal diagnose")
	writeSkill(t, dir, "premarket_stock", "Cron stock premarket")
	l := procedural.NewLoader(dir)
	cat := l.ChatCatalog()
	ids := map[string]struct{}{}
	for _, e := range cat {
		ids[e.ID] = struct{}{}
	}
	if _, ok := ids["signal_diagnose"]; !ok {
		t.Fatal("expected signal_diagnose in catalog")
	}
	if _, ok := ids["premarket_stock"]; ok {
		t.Fatal("premarket_stock should not be in chat catalog")
	}
}

func TestFormatCatalogNonEmpty(t *testing.T) {
	text := procedural.FormatCatalog([]procedural.CatalogEntry{
		{ID: "signal_diagnose", Description: "诊断信号"},
	})
	if text == "" || !contains(text, "skill_view") || !contains(text, "signal_diagnose") {
		t.Fatalf("catalog format: %q", text)
	}
}

func TestReadSkillFileSibling(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\ndescription: d\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "extra.md"), []byte("extra content"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := procedural.NewLoader(dir)
	_, body, err := l.ReadSkillFile("demo", "extra.md")
	if err != nil || body != "extra content" {
		t.Fatalf("ReadSkillFile: body=%q err=%v", body, err)
	}
}

func writeSkill(t *testing.T, root, name, desc string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "---\nname: " + name + "\ndescription: " + desc + "\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub)))
}

func containsHelper(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
