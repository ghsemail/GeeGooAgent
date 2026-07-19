package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/inspect"
)

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	quick := fs.Bool("quick", false, "run geegoo verify agent-loop cards")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = application.Close() }()

	report := inspect.Build(application, inspect.Options{
		ConfigPath: *configPath,
		QuickLoop:  *quick,
	})
	fmt.Print(inspect.FormatText(report))
}
