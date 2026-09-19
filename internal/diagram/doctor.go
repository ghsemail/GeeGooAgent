package diagram

import (
	"fmt"
	"os"
	"path/filepath"
)

// Check is a doctor row produced by this module (mapped by the doctor package).
type Check struct {
	Name   string
	OK     bool
	Warn   bool
	Detail string
}

// Checks reports optional Node availability and artifact coverage.
// Missing Node is a warning, never a hard failure.
func Checks(projectRoot string) []Check {
	root := projectRoot
	if root == "" {
		root = "."
	}
	svc := New(root)
	infos := svc.List()
	missing := 0
	for _, info := range infos {
		if _, err := os.Stat(artifactPath(root, info.ID)); err != nil {
			missing++
		}
	}
	artifact := Check{
		Name:   "diagram_artifacts",
		OK:     true,
		Warn:   missing > 0,
		Detail: fmt.Sprintf("%d/%d cached HTML under %s", len(infos)-missing, len(infos), filepath.ToSlash(ArtifactDir)),
	}
	if missing > 0 {
		artifact.Detail += " — run go generate ./internal/diagram"
	}

	cli := ArchifyCLI{Root: root}
	node := Check{Name: "diagram_archify", OK: true, Warn: true}
	switch {
	case cli.Available():
		node.Warn = false
		node.Detail = "node + third_party/archify available (optional generate-time renderer)"
	case NodeAvailable():
		node.Detail = "node found; third_party/archify not installed (builtin renderer in use)"
	default:
		node.Detail = "node not on PATH; runtime serves cached/builtin HTML"
	}
	return []Check{artifact, node}
}
