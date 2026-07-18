package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/verify"
)

func runVerifyAgentLoop(args []string) {
	fs := flag.NewFlagSet("verify agent-loop", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(*configPath, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify agent-loop: %v\n", err)
		os.Exit(2)
	}
	defer application.Close()

	cards := verify.VerifyAgentLoopParity(application.Registry)
	if application.Config.ProfileFeatureEnabled() {
		fmt.Printf("Profile: %s\n", application.Config.ProfileSummary())
	}
	for _, c := range cards {
		mark := "✓"
		if !c.Passed {
			mark = "✗"
		}
		fmt.Printf("  %s %s\n", mark, c.Summary())
	}
	if !verify.AllAgentLoopPass(cards) {
		os.Exit(1)
	}
	fmt.Println("\nAgent loop parity: PASS")
}
