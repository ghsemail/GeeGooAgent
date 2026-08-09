//go:build ignore

// check_import_boundaries verifies Agent OS package dependency rules (P5).
// Usage: go run scripts/check_import_boundaries.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghsemail/GeeGooAgent/internal/archboundaries"
)

func main() {
	root := filepath.Join("internal")
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	violations, err := archboundaries.Check(root, archboundaries.DefaultRules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check failed: %v\n", err)
		os.Exit(2)
	}
	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "import boundary violations:\n")
		for _, v := range violations {
			fmt.Fprintf(os.Stderr, "  - %s\n", v)
		}
		os.Exit(1)
	}
	fmt.Println("import boundaries OK")
}
