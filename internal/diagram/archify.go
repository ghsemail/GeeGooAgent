package diagram

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ArchifyCLI renders via `node third_party/archify/bin/archify.mjs deliver`.
type ArchifyCLI struct {
	Root    string
	Node    string
	Timeout time.Duration
}

func (a ArchifyCLI) Name() string { return "archify" }

func (a ArchifyCLI) Available() bool {
	bin := a.bin()
	if _, err := os.Stat(bin); err != nil {
		return false
	}
	node := a.nodeBin()
	if node == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-v")
	return cmd.Run() == nil
}

func (a ArchifyCLI) Render(ctx context.Context, doc Document) (Artifact, error) {
	if !a.Available() {
		return Artifact{}, fmt.Errorf("diagram: archify CLI not available")
	}
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "geegoo-diagram-*")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(dir)

	kind := doc.Kind
	if kind != KindArchitecture {
		kind = KindWorkflow
	}
	in := filepath.Join(dir, "source.json")
	out := filepath.Join(dir, "artifact.html")
	if err := os.WriteFile(in, doc.IR, 0o644); err != nil {
		return Artifact{}, err
	}
	args := []string{a.bin(), "deliver", kind, in, out, "--quality", "standard", "--json"}
	cmd := exec.CommandContext(ctx, a.nodeBin(), args...)
	cmd.Dir = filepath.Dir(a.bin())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String() + "\n" + stdout.String())
		return Artifact{}, fmt.Errorf("diagram: archify deliver: %w: %s", err, msg)
	}
	html, err := os.ReadFile(out)
	if err != nil {
		return Artifact{}, err
	}
	if len(html) == 0 {
		return Artifact{}, fmt.Errorf("diagram: archify produced empty HTML")
	}
	sum := sha256.Sum256(html)
	return Artifact{
		HTML:        html,
		SHA256:      hex.EncodeToString(sum[:]),
		ContentType: "text/html; charset=utf-8",
		Source:      "archify",
	}, nil
}

func (a ArchifyCLI) bin() string {
	root := strings.TrimSpace(a.Root)
	if root == "" {
		root = "."
	}
	return filepath.Join(root, "third_party", "archify", "bin", "archify.mjs")
}

func (a ArchifyCLI) nodeBin() string {
	if v := strings.TrimSpace(a.Node); v != "" {
		return v
	}
	if p, err := exec.LookPath("node"); err == nil {
		return p
	}
	return ""
}

// NodeAvailable reports whether a node binary is on PATH.
func NodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}
