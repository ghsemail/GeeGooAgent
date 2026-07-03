package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func runScheduler(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: geegoo scheduler <run|list> [flags]")
		os.Exit(2)
	}
	switch args[0] {
	case "run":
		runSchedulerRun(args[1:])
	case "list":
		runSchedulerList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "geegoo scheduler: unknown subcommand %q (try: run, list)\n", args[0])
		os.Exit(2)
	}
}

func runSchedulerRun(args []string) {
	fs := flag.NewFlagSet("scheduler run", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
		os.Exit(2)
	}
	defer application.Close()
	workspace, _ := application.Config.ResolveOutputDir()
	jobsDir := filepath.Join(workspace, "scheduler")
	runner := scheduler.NewRunner(application, jobsDir)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("geegoo scheduler started; Ctrl+C to stop.")
	if err := runner.Start(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
		os.Exit(1)
	}
}

func runSchedulerList(args []string) {
	fs := flag.NewFlagSet("scheduler list", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scheduler list: %v\n", err)
		os.Exit(2)
	}
	workspace, _ := cfg.ResolveOutputDir()
	jobsDir := filepath.Join(workspace, "scheduler")
	jf, err := scheduler.LoadJobs(jobsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scheduler list: %v\n", err)
		os.Exit(1)
	}
	if len(jf.Jobs) == 0 {
		fmt.Println("(no jobs; defaults would be: pre_market_weekday @ 0 8 * * 1-5)")
		return
	}
	fmt.Printf("%-22s  %-12s  %-14s  %-8s  %-16s  %s\n",
		"NAME", "SKILL", "CRON", "STATE", "VERDICT", "LAST RUN")
	for _, j := range jf.Jobs {
		fmt.Println(scheduler.FormatJob(j))
	}
}
