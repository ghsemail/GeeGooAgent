package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func runSkill(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "usage: geegoo run <skill>")
		os.Exit(2)
	}
	skill := fs.Args()[0]

	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(2)
	}
	if application.Gateway == nil && !*dryRun {
		fmt.Fprintln(os.Stderr, "warning: LLM not configured (workflow stub mode)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := application.RunSkillContext(ctx, skill)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("session=%s status=%s\n", result.SessionID, result.Status)
	if result.LastError != "" {
		fmt.Fprintf(os.Stderr, "error=%s\n", result.LastError)
	}
	if !result.OK() {
		os.Exit(1)
	}
}
