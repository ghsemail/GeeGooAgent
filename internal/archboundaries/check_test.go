package archboundaries_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/archboundaries"
)

func TestImportBoundariesClean(t *testing.T) {
	root := filepath.Join("..") // internal/ when cwd is internal/archboundaries
	violations, err := archboundaries.Check(root, archboundaries.DefaultRules)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("import boundary violations:\n%s", stringsJoin(violations))
	}
}

func stringsJoin(lines []string) string {
	out := ""
	for _, l := range lines {
		out += "  - " + l + "\n"
	}
	return out
}
