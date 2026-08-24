package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/doctor"
)

const cliName = "geegoo"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "setup":
		runSetup(os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	case "resume":
		runResume(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
	case "inspect":
		runInspect(os.Args[2:])
	case "exec":
		runExec(os.Args[2:])
	case "mcp":
		runMCP(os.Args[2:])
	case "chat":
		runChat(os.Args[2:])
	case "run":
		runSkill(os.Args[2:])
	case "notify-stock":
		notifyStock(os.Args[2:])
	case "migrate":
		runMigrate(os.Args[2:])
	case "skills":
		runSkills(os.Args[2:])
	case "skill":
		runSkillManage(os.Args[2:])
	case "scheduler":
		runScheduler(os.Args[2:])
	case "gateway":
		runGateway(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
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
	skipLLM := fs.Bool("skip-llm", false, "skip LLM ping")
	skipAPI := fs.Bool("skip-api", false, "skip MCP API ping")
	_ = skipLLM
	_ = skipAPI
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	os.Exit(doctor.Run(*configPath))
}

func printUsage() {
	fmt.Printf(`%s - GeeGoo Agent CLI (Go)

Usage:
  %s setup [--config PATH] [--force]
  %s update [--dir PATH]
  %s resume --session ID [--config PATH] [--dry-run]
  %s doctor [--config PATH]
  %s inspect [--config PATH] [--quick] [--session ID]
  %s exec -p "..." [--config PATH] [--session ID] [--output-format ndjson|text] [--dry-run]
  %s mcp serve [--config PATH] [--toolset chat|workflow] [--dry-run]
  %s chat [--config PATH] [--dry-run] [--message TEXT] [--output-format text|ndjson] [--session ID] [--tui|--cli]
  %s run <skill> [--config PATH] [--dry-run] [--no-notify-feishu]
  %s notify-stock --session ID --skill premarket_stock|postmarket_stock --market CN|HK|US
  %s migrate [--config PATH] [--from file|sqlite] [--to sqlite|postgres] [--dry-run]
  %s migrate tenant --assign-user UID [--session ID] [--compare-from PATH] [--dry-run]
  %s skills list [--config PATH]
  %s skill enable|disable <name> [--config PATH]
  %s scheduler <run|list> [--config PATH]
  %s gateway <run|status|setup> [--config PATH] [--dry-run]
  %s verify --codes CODES [--date YYYY-MM-DD] [--config PATH]
  %s verify agent-loop [--config PATH]

`, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName, cliName)
}
