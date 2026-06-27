package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func runChat(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	message := fs.String("message", "", "single-turn message (non-interactive)")
	sessionID := fs.String("session", "", "resume existing chat session id")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(2)
	}
	if application.Gateway == nil {
		fmt.Fprintf(os.Stderr, "chat: LLM 未配置，请填写 llm.token_key\n")
		os.Exit(2)
	}

	repl, err := chatrepl.NewWithSession(application, *configPath, *sessionID, *dryRun, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(2)
	}
	if *message != "" {
		os.Exit(repl.RunSingle(*message))
	}
	os.Exit(repl.Run())
}
