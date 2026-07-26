package searchrunner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools/searchrunner"
)

func TestWebSearchUsesBundledScript(t *testing.T) {
	t.Parallel()
	root := findRepoRoot(t)
	if root == "" {
		t.Skip("repo root not found")
	}
	script := filepath.Join(root, "skills", "bundled", "duckduckgo-search", "scripts", "web_search.py")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("bundled script missing: %v", err)
	}
	if os.Getenv("GEEGOO_RUN_NETWORK_TESTS") != "1" {
		t.Skip("set GEEGOO_RUN_NETWORK_TESTS=1 to run live DuckDuckGo integration test")
	}
	hits, err := searchrunner.WebSearch(t.Context(), searchrunner.Options{
		ProjectRoot: root,
		BundledOnly: true,
	}, "SpaceX", 2)
	if err != nil {
		t.Fatalf("WebSearch: %v", err)
	}
	if len(hits) == 0 {
		t.Skip("duckduckgo returned no results (network/rate limit)")
	}
	if hits[0].Title == "" {
		t.Fatalf("expected title in first hit: %+v", hits[0])
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "skills", "bundled", "duckduckgo-search", "scripts", "web_search.py")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
