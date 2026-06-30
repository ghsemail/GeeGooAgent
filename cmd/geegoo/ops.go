package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	force := fs.Bool("force", false, "overwrite existing config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if _, err := os.Stat(*configPath); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "setup: config already exists: %s (use --force to overwrite)\n", *configPath)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*configPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "setup: create config directory: %v\n", err)
		os.Exit(1)
	}
	cfg := defaultConfig()
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: encode config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*configPath, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "setup: write config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("config=%s\n", *configPath)
	fmt.Printf("mcp=%s signal_catalog=%s signal_analyze=%s data=%s\n",
		config.DefaultBotMCPURL,
		config.DefaultSignalCatalogURL,
		config.DefaultSignalAnalyzeURL,
		config.DefaultDataHTTPURL,
	)
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	dir := fs.String("dir", ".", "project directory")
	output := fs.String("output", defaultBinaryPath(), "compiled binary path")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := runCommand(*dir, "git", "pull", "--ff-only"); err != nil {
		fmt.Fprintf(os.Stderr, "update: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "update: create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := runCommand(*dir, "go", "build", "-o", *output, "./cmd/geegoo"); err != nil {
		fmt.Fprintf(os.Stderr, "update: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("built=%s\n", *output)
}

func runResume(args []string) {
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	sessionID := fs.String("session", "", "workflow session id")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *sessionID == "" {
		fmt.Fprintln(os.Stderr, "resume: --session is required")
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resume: %v\n", err)
		os.Exit(2)
	}
	result, err := application.ResumePreMarket(*sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resume: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("session=%s status=%s\n", result.SessionID, result.Status)
	if result.LastError != "" {
		fmt.Fprintf(os.Stderr, "error=%s\n", result.LastError)
	}
	if !result.OK() {
		os.Exit(1)
	}
}

func defaultConfig() config.AppConfig {
	return config.AppConfig{
		BaseURL:       config.DefaultBotMCPURL,
		GeeGooURL:     config.DefaultBotMCPURL,
		SignalBaseURL: config.DefaultSignalCatalogURL,
		DataBaseURL:   config.DefaultDataHTTPURL,
		OutputDir:     filepath.Join(config.Home(), "data"),
		MaxSteps:      80,
		LLM: config.LLMConfig{
			Provider:        "openai",
			TokenKey:        "",
			Model:           "gpt-4.1-mini",
			Temperature:     0.2,
			MaxTokens:       4096,
			ReasoningEffort: "medium",
		},
		Search:  config.SearchConfig{Provider: "duckduckgo", MaxResults: 5},
		Sandbox: config.SandboxConfig{AllowedHosts: config.DefaultAllowedHosts()},
	}
}

func defaultBinaryPath() string {
	name := "geegoo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join("bin", name)
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}
