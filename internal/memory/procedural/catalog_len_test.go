package procedural_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
)

func TestRepoSkillsCatalogNonEmpty(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills")
	l := procedural.NewLoader(root)
	if n := len(l.ChatCatalog()); n < 5 {
		t.Fatalf("expected chat catalog entries, got %d", n)
	}
}
