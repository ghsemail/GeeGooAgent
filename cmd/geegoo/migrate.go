package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func runMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "preview source rows without writing SQLite")
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

	fileState := infra.NewStateStore(workspace)
	fileStore := chatsession.NewChatSessionStore(fileState)
	ids, err := fileStore.ListSessionIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: list file sessions: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("源（文件存储）会话数: %d\n", len(ids))
	if *dryRun {
		fmt.Println("--dry-run: 不写入 SQLite，仅预览。")
		for _, id := range ids {
			s, err := fileStore.Load(id)
			if err != nil || s == nil {
				continue
			}
			fmt.Printf("  %s  %s  status=%s  msgs=%d\n", s.ID, s.Title, s.Status, len(s.Messages))
		}
		return
	}

	dbPath := filepath.Join(workspace, "geegoo.db")
	db, err := infra.OpenSQLite(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: open sqlite: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	sqlStore := chatsession.NewSQLiteSessionStore(db)

	migrated, skipped := 0, 0
	for _, id := range ids {
		s, err := fileStore.Load(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  跳过 %s: 加载失败 %v\n", id, err)
			skipped++
			continue
		}
		if s == nil {
			skipped++
			continue
		}
		if err := sqlStore.Save(s); err != nil {
			fmt.Fprintf(os.Stderr, "  跳过 %s: 保存失败 %v\n", id, err)
			skipped++
			continue
		}
		migrated++
	}
	fmt.Printf("迁移完成: 写入 %d，跳过 %d\n", migrated, skipped)
	fmt.Printf("SQLite: %s\n", dbPath)

	// Verify count.
	verify, _ := sqlStore.ListSessionIDs()
	if len(verify) < migrated {
		fmt.Fprintf(os.Stderr, "警告: 校验失败，SQLite 仅有 %d 条\n", len(verify))
		os.Exit(1)
	}
	_ = app.App{}
}
