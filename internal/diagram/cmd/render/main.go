// Command render compiles catalog IRs and writes HTML artifacts.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/diagram"
)

func main() {
	root := "."
	if v := os.Getenv("GEEGOO_PROJECT_ROOT"); v != "" {
		root = v
	}
	root = diagram.FindRepoRoot(root)
	svc := diagram.New(root)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	n := 0
	for _, info := range svc.List() {
		doc, err := svc.Get(info.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compile %s: %v\n", info.ID, err)
			os.Exit(1)
		}
		art, err := svc.RenderDocument(ctx, doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "render %s: %v\n", info.ID, err)
			os.Exit(1)
		}
		if art.Source != "archify" && svc.Archify != nil {
			if _, aerr := svc.Archify.Render(ctx, doc); aerr != nil {
				fmt.Fprintf(os.Stderr, "archify fallback %s: %v\n", info.ID, aerr)
			}
		}
		if err := diagram.WriteArtifact(root, info.ID, art); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", info.ID, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%s, %d bytes)\n", info.ID, art.Source, len(art.HTML))
		n++
	}
	fmt.Printf("rendered %d diagrams into %s\n", n, filepath.Join(root, diagram.ArtifactDir))
}
