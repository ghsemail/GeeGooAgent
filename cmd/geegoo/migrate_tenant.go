package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
)

func runMigrateTenant(args []string) {
	fs := flag.NewFlagSet("migrate tenant", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	assignUser := fs.String("assign-user", "", "user id to stamp on orphan sessions")
	sessionID := fs.String("session", "", "single session id to assign (optional)")
	compareLegacy := fs.String("compare-from", "", "legacy global compare/history.jsonl to copy")
	dryRun := fs.Bool("dry-run", false, "preview only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	userID := strings.TrimSpace(*assignUser)
	if userID == "" {
		fmt.Fprintln(os.Stderr, "migrate tenant: --assign-user required")
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate tenant: %v\n", err)
		os.Exit(2)
	}
	workspace, err := cfg.ResolveOutputDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate tenant: workspace: %v\n", err)
		os.Exit(2)
	}

	store, cleanup, err := openTenantSessionStore(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate tenant: store: %v\n", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	if sid := strings.TrimSpace(*sessionID); sid != "" {
		if *dryRun {
			fmt.Printf("would assign session %s -> user %s\n", sid, userID)
		} else {
			s, loadErr := store.Load(sid)
			if loadErr != nil || s == nil {
				fmt.Fprintf(os.Stderr, "session %s not found\n", sid)
				os.Exit(1)
			}
			chatsession.SetUserID(s, userID)
			if err := store.Save(s); err != nil {
				fmt.Fprintf(os.Stderr, "save: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("assigned session %s -> user %s\n", sid, userID)
		}
	} else {
		if *dryRun {
			entries, _ := store.ListIndexedSessions()
			orphan := 0
			for _, e := range entries {
				if chatsession.UserIDFromEntry(e) == "" {
					orphan++
					fmt.Printf("  orphan session %s\n", e.ID)
				}
			}
			fmt.Printf("would stamp %d orphan session(s) -> user %s\n", orphan, userID)
		} else {
			n, err := chatsession.StampOrphanSessions(store, userID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "stamp sessions: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("stamped %d orphan session(s) -> user %s\n", n, userID)
		}
	}

	legacy := strings.TrimSpace(*compareLegacy)
	if legacy == "" {
		legacy = filepath.Join(workspace, "compare", "history.jsonl")
	}
	if *dryRun {
		if _, err := os.Stat(legacy); err == nil {
			fmt.Printf("would copy compare history from %s -> user %s\n", legacy, userID)
		}
	} else {
		n, err := runtimeapi.MigrateLegacyCompareHistory(workspace, userID, legacy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compare migrate: %v\n", err)
			os.Exit(1)
		}
		if n > 0 {
			fmt.Printf("copied %d compare run(s) for user %s\n", n, userID)
		}
	}
}

func openTenantSessionStore(workspace string) (chatsession.SessionStore, func(), error) {
	if dsn := infra.PostgresDSN(); dsn != "" {
		pg, err := infra.OpenPostgres(dsn)
		if err != nil {
			return nil, nil, err
		}
		return chatsession.NewPostgresSessionStore(pg.SQL()), func() { _ = pg.Close() }, nil
	}
	dbPath := filepath.Join(workspace, "geegoo.db")
	db, err := infra.OpenSQLite(dbPath)
	if err != nil {
		return nil, nil, err
	}
	return chatsession.NewSQLiteSessionStore(db), func() { _ = db.Close() }, nil
}
