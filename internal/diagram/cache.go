package diagram

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactDir is the default on-disk cache relative to the repo root.
const ArtifactDir = "internal/diagram/artifacts"

// FindRepoRoot walks up from start (or cwd) until go.mod is found.
func FindRepoRoot(start string) string {
	dir := strings.TrimSpace(start)
	if dir == "" {
		dir = "."
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	cur := dir
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dir
}

func artifactPath(root, id string) string {
	return filepath.Join(root, ArtifactDir, safeFileName(id)+".html")
}

func hashPath(root, id string) string {
	return filepath.Join(root, ArtifactDir, safeFileName(id)+".hash")
}

func safeFileName(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}

func loadArtifact(root, id string) (Artifact, bool) {
	raw, err := os.ReadFile(artifactPath(root, id))
	if err != nil || len(raw) == 0 {
		return Artifact{}, false
	}
	sum := sha256.Sum256(raw)
	got := hex.EncodeToString(sum[:])
	if want, err := os.ReadFile(hashPath(root, id)); err == nil {
		if strings.TrimSpace(string(want)) != "" && strings.TrimSpace(string(want)) != got {
			return Artifact{}, false
		}
	}
	return Artifact{
		HTML:        raw,
		SHA256:      got,
		ContentType: "text/html; charset=utf-8",
		Source:      "artifact",
	}, true
}

// WriteArtifact stores HTML + sha256 sidecar for a catalog id.
func WriteArtifact(root, id string, art Artifact) error {
	return writeArtifact(root, id, art)
}

func writeArtifact(root, id string, art Artifact) error {
	if err := os.MkdirAll(filepath.Join(root, ArtifactDir), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(artifactPath(root, id), art.HTML, 0o644); err != nil {
		return err
	}
	return os.WriteFile(hashPath(root, id), []byte(art.SHA256+"\n"), 0o644)
}
