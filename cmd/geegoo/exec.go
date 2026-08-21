package main

import (
	"flag"
	"fmt"
	"os"
)

func runExec(args []string) {
	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	prompt := fs.String("p", "", "prompt text (alias for --message)")
	message := fs.String("message", "", "single-turn message")
	configPath := fs.String("config", "", "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	sessionID := fs.String("session", "", "resume existing chat session id")
	outputFormat := fs.String("output-format", "ndjson", "output format: ndjson|text")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	text := *message
	if text == "" {
		text = *prompt
	}
	if text == "" {
		fmt.Fprintf(os.Stderr, "exec: -p or --message required\n")
		os.Exit(2)
	}
	chatArgs := []string{
		"--message", text,
		"--output-format", *outputFormat,
	}
	if *configPath != "" {
		chatArgs = append(chatArgs, "--config", *configPath)
	}
	if *dryRun {
		chatArgs = append(chatArgs, "--dry-run")
	}
	if *sessionID != "" {
		chatArgs = append(chatArgs, "--session", *sessionID)
	}
	runChat(chatArgs)
}
