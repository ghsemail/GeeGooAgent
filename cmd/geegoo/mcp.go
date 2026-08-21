package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/mcpserver"
)

func runMCP(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: geegoo mcp serve [--config PATH] [--toolset chat|workflow]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "serve":
		runMCPServe(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "geegoo mcp: unknown subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runMCPServe(args []string) {
	fs := flag.NewFlagSet("mcp serve", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	toolset := fs.String("toolset", "chat", "toolset: chat|workflow")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp serve: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = application.Close() }()
	names := application.ChatToolNames()
	if *toolset == "workflow" {
		names = application.Registry.ListNames()
	}
	if err := mcpserver.ServeStdio(application, names); err != nil {
		fmt.Fprintf(os.Stderr, "mcp serve: %v\n", err)
		os.Exit(1)
	}
}
