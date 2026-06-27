package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func runChat(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	message := fs.String("message", "", "single-turn message (non-interactive)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(2)
	}

	session := runtime.NewSession()
	schemas := application.Registry.Schemas(tools.ChatToolNames)
	toolCtx := application.ToolContext(session.ID)

	fmt.Printf("geegoo chat (Go) — %s\n", application.EndpointSummary())
	fmt.Printf("session=%s  输入 /exit 退出\n\n", session.ID)

	if *message != "" {
		result := application.Loop.RunTurn(session, *message, toolCtx, schemas)
		printTurnResult(result)
		if result.Failed {
			os.Exit(1)
		}
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			break
		}
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if text == "/exit" || text == "/quit" {
			break
		}
		result := application.Loop.RunTurn(session, text, toolCtx, schemas)
		printTurnResult(result)
		fmt.Println()
	}
}

func printTurnResult(result runtime.TurnResult) {
	fmt.Println(result.AssistantText)
	if result.Failed && result.Error != "" {
		fmt.Fprintf(os.Stderr, "error: %s\n", result.Error)
	}
}
