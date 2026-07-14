package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chattui"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func runChat(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	message := fs.String("message", "", "single-turn message (non-interactive)")
	sessionID := fs.String("session", "", "resume existing chat session id")
	forceTUI := fs.Bool("tui", false, "force Bubble Tea TUI")
	forceCLI := fs.Bool("cli", false, "force classic CLI (go-prompt)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = application.Close() }()
	if application.Gateway == nil {
		fmt.Fprintf(os.Stderr, "chat: LLM 未配置，请填写 llm.token_key\n")
		os.Exit(2)
	}

	if *message != "" {
		repl, err := chatrepl.NewWithSession(application, *configPath, *sessionID, *dryRun, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chat: %v\n", err)
			os.Exit(2)
		}
		os.Exit(repl.RunSingle(*message))
	}

	if chattui.ShouldUseTUI(application.Config, *forceTUI, *forceCLI) {
		os.Exit(chattui.Run(chattui.RunOpts{
			App: application, ConfigPath: *configPath, SessionID: *sessionID, DryRun: *dryRun,
		}))
	}

	repl, err := chatrepl.NewWithSession(application, *configPath, *sessionID, *dryRun, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(2)
	}
	os.Exit(repl.Run())
}
