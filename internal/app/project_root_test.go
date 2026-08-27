package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRootFromInstallDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "geegoo-agent")
	mustMkdir(t, filepath.Join(root, "skills", "premarket_market"))
	if err := os.WriteFile(filepath.Join(root, "skills", "premarket_market", "SKILL.md"), []byte("---\nname: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GEEGOO_PROJECT_ROOT", "")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	t.Cleanup(func() { _ = os.Chdir(wd) })

	t.Setenv("GEEGOO_HOME", filepath.Dir(root))
	got := findProjectRoot()
	if got != root {
		t.Fatalf("findProjectRoot()=%q want %q", got, root)
	}
}

func TestFindProjectRootHonorsEnvOverride(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(root, "skills", "premarket_market"))
	if err := os.WriteFile(filepath.Join(root, "skills", "premarket_market", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GEEGOO_PROJECT_ROOT", root)
	if got := findProjectRoot(); got != root {
		t.Fatalf("findProjectRoot()=%q want %q", got, root)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
