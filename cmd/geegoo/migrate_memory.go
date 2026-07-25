package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// runMigrateMemoryChunks backfills agent_memory_chunks from persisted session summaries.
func runMigrateMemoryChunks(args []string) {
	fs := flag.NewFlagSet("migrate memory-chunks", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config.json")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	path := strings.TrimSpace(*configPath)
	if path == "" {
		path = os.Getenv("GEEGOO_CONFIG")
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "migrate memory-chunks: set --config or GEEGOO_CONFIG")
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(path, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate memory-chunks: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()
	if application.Semantic == nil {
		fmt.Fprintln(os.Stderr, "migrate memory-chunks: semantic memory not enabled (GEEGOO_PG_DSN + GEEGOO_VECTOR_ENABLE)")
		os.Exit(1)
	}
	store, err := application.SessionStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate memory-chunks: session store: %v\n", err)
		os.Exit(1)
	}
	ids, err := store.ListSessionIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate memory-chunks: list sessions: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()
	written, skipped := 0, 0
	for _, id := range ids {
		sess, err := store.Load(id)
		if err != nil || sess == nil {
			skipped++
			continue
		}
		summary := strings.TrimSpace(sess.Summary)
		if summary == "" {
			skipped++
			continue
		}
		userID := chatsession.UserIDFromSession(sess)
		if *dryRun {
			fmt.Printf("  %s  user=%s  summary=%d chars\n", id, userID, len(summary))
			written++
			continue
		}
		if err := application.Semantic.UpsertSummary(ctx, sess.ID, userID, summary); err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", id, err)
			skipped++
			continue
		}
		written++
	}
	fmt.Printf("memory-chunks: upserted %d, skipped %d (of %d sessions)\n", written, skipped, len(ids))
}
