package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/inspect"
)

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	quick := fs.Bool("quick", false, "run geegoo verify agent-loop cards")
	sessionID := fs.String("session", "", "show compaction lineage for a chat session id")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = application.Close() }()

	if strings.TrimSpace(*sessionID) != "" {
		store, err := application.SessionStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
			os.Exit(2)
		}
		chat, err := store.Load(strings.TrimSpace(*sessionID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
			os.Exit(2)
		}
		if chat == nil {
			fmt.Fprintf(os.Stderr, "inspect: session not found: %s\n", *sessionID)
			os.Exit(2)
		}
		fmt.Print(inspect.FormatSessionText(inspect.BuildSession(chat)))
		return
	}

	report := inspect.Build(application, inspect.Options{
		ConfigPath: *configPath,
		QuickLoop:  *quick,
	})
	fmt.Print(inspect.FormatText(report))
}
