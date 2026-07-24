package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func runMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	target := fs.String("to", "sqlite", "target store: sqlite|postgres")
	source := fs.String("from", "file", "source store: file|sqlite")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(2)
	}
	workspace, err := cfg.ResolveOutputDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: resolve workspace: %v\n", err)
		os.Exit(2)
	}

	src, err := openSessionStore(*source, workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: source: %v\n", err)
		os.Exit(1)
	}
	ids, err := src.ListSessionIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: list sessions: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("源（%s）会话数: %d → 目标 %s\n", *source, len(ids), *target)
	if *dryRun {
		for _, id := range ids {
			s, _ := src.Load(id)
			if s == nil {
				continue
			}
			fmt.Printf("  %s  %s  msgs=%d\n", s.ID, s.Title, len(s.Messages))
		}
		return
	}

	dest, cleanup, err := openSessionStoreTarget(*target, workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: target: %v\n", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	migrated, skipped := 0, 0
	for _, id := range ids {
		s, err := src.Load(id)
		if err != nil || s == nil {
			skipped++
			continue
		}
		if err := dest.Save(s); err != nil {
			fmt.Fprintf(os.Stderr, "  跳过 %s: %v\n", id, err)
			skipped++
			continue
		}
		migrated++
	}
	fmt.Printf("迁移完成: 写入 %d，跳过 %d\n", migrated, skipped)
}

func openSessionStore(kind, workspace string) (chatsession.SessionStore, error) {
	switch kind {
	case "file":
		return chatsession.NewChatSessionStore(infra.NewStateStore(workspace)), nil
	case "sqlite":
		dbPath := filepath.Join(workspace, "geegoo.db")
		db, err := infra.OpenSQLite(dbPath)
		if err != nil {
			return nil, err
		}
		return chatsession.NewSQLiteSessionStore(db), nil
	default:
		return nil, fmt.Errorf("unknown source %q", kind)
	}
}

func openSessionStoreTarget(kind, workspace string) (chatsession.SessionStore, func(), error) {
	switch kind {
	case "sqlite":
		dbPath := filepath.Join(workspace, "geegoo.db")
		db, err := infra.OpenSQLite(dbPath)
		if err != nil {
			return nil, nil, err
		}
		return chatsession.NewSQLiteSessionStore(db), func() { _ = db.Close() }, nil
	case "postgres":
		dsn := infra.PostgresDSN()
		if dsn == "" {
			return nil, nil, fmt.Errorf("GEEGOO_PG_DSN not set")
		}
		pg, err := infra.OpenPostgres(dsn)
		if err != nil {
			return nil, nil, err
		}
		return chatsession.NewPostgresSessionStore(pg.SQL()), func() { _ = pg.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("unknown target %q", kind)
	}
}
