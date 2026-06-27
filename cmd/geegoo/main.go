package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/doctor"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const cliName = "geegoo"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "doctor":
		runDoctor(os.Args[2:])
	case "chat":
		runChat(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command %q\n", cliName, os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	skipLLM := fs.Bool("skip-llm", false, "skip LLM ping (A0 default)")
	skipAPI := fs.Bool("skip-api", false, "skip MCP API ping (A0 default)")
	_ = skipLLM
	_ = skipAPI
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	os.Exit(doctor.Run(*configPath))
}

func printUsage() {
	fmt.Printf(`%s — GeeGoo Agent CLI (Go)

Usage:
  %s doctor [--config PATH]
  %s chat [--config PATH] [--dry-run] [--message TEXT]

Subcommands (planned):
  setup, update, run, resume

`, cliName, cliName, cliName)
}
