// Package archboundaries verifies Agent OS import dependency rules (P5).
package archboundaries

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Rule forbids a package from importing any of the listed import path suffixes.
type Rule struct {
	Package string
	Forbid  []string
}

// DefaultRules match docs/architecture/agent-runtime-architecture.md §5.
var DefaultRules = []Rule{
	{
		Package: "github.com/ghsemail/GeeGooAgent/internal/cognition",
		Forbid: []string{
			"/internal/agent", "/internal/cli", "/internal/runtimeapi",
			"/internal/tools", "/internal/app",
		},
	},
	{
		Package: "github.com/ghsemail/GeeGooAgent/internal/tools",
		Forbid:  []string{"/internal/cognition"},
	},
	{
		Package: "github.com/ghsemail/GeeGooAgent/internal/memport",
		Forbid: []string{
			"/internal/memory", "/internal/tools", "/internal/agent",
		},
	},
	{
		Package: "github.com/ghsemail/GeeGooAgent/internal/infra",
		Forbid: []string{
			"/internal/runtime", "/internal/tools", "/internal/llm", "/internal/agent",
		},
	},
}

// Check walks internal/ Go sources and returns violations.
func Check(root string, rules []Rule) ([]string, error) {
	if root == "" {
		root = "internal"
	}
	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		pkgPath, imports, err := parseImports(path)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if pkgPath != rule.Package {
				continue
			}
			for _, imp := range imports {
				for _, bad := range rule.Forbid {
					if strings.Contains(imp, bad) {
						violations = append(violations, fmt.Sprintf("%s imports forbidden %s (rule %s)", path, imp, bad))
					}
				}
			}
		}
		return nil
	})
	return violations, err
}

func parseImports(path string) (pkgPath string, imports []string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return "", nil, err
	}
	if f.Name != nil {
		// package name only; map dir to module path below
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	pkgPath = "github.com/ghsemail/GeeGooAgent/" + strings.TrimPrefix(dir, "internal/")
	pkgPath = strings.ReplaceAll(pkgPath, "//", "/")
	for _, imp := range f.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}
	return pkgPath, imports, nil
}
