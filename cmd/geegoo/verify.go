package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/verify"
)

func runVerify(args []string) {
	if len(args) > 0 && args[0] == "agent-loop" {
		runVerifyAgentLoop(args[1:])
		return
	}
	runVerifyReports(args)
}

func runVerifyReports(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	date := fs.String("date", "", "report date YYYY-MM-DD (default today)")
	codesFlag := fs.String("codes", "", "comma-separated stock codes to verify")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *date == "" {
		*date = todayDate()
	}
	if *codesFlag == "" {
		fmt.Fprintln(os.Stderr, "verify: --codes is required, e.g. --codes 00700.HK,000001.SZ")
		os.Exit(2)
	}
	codes := splitCodes(*codesFlag)

	application, err := app.LoadFromConfigPath(*configPath, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify: %v\n", err)
		os.Exit(2)
	}
	defer application.Close()

	ctx := context.Background()
	var allCards []verify.ReportCard
	for _, code := range codes {
		reports, err := application.MCP.GetStockDailyReports(ctx, application.Config.MCPToken(), code, *date)
		if err != nil {
			fmt.Fprintf(os.Stderr, "verify: %s: %v\n", code, err)
			continue
		}
		cards := verify.VerifyReports(reports.PreMarket)
		allCards = append(allCards, cards...)
		for _, c := range cards {
			fmt.Println(c.Summary())
			for _, ch := range c.Checks {
				mark := "✓"
				if !ch.Passed {
					mark = "✗"
				}
				fmt.Printf("    %s %-16s %s\n", mark, ch.Name, ch.Detail)
			}
		}
	}

	fmt.Println("\nField completeness matrix:")
	matrix := verify.CompletenessMatrix(allCards)
	for name, rate := range matrix {
		fmt.Printf("  %-16s %.0f%%\n", name, rate*100)
	}

	if len(allCards) == 0 {
		fmt.Println("(no pre_market reports found for the given date/codes)")
		os.Exit(1)
	}
	if !verify.AllPass(allCards) {
		os.Exit(1)
	}
}

func splitCodes(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}
