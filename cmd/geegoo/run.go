package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func runSkill(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip mutating API calls")
	code := fs.String("code", "", "intraday: stock code (e.g. 00700.HK)")
	stockName := fs.String("stock-name", "", "intraday: stock name")
	botID := fs.String("bot-id", "", "intraday: bot id")
	botName := fs.String("bot-name", "", "intraday: bot name")
	botType := fs.String("bot-type", "", "intraday: bot type (DCA/GRID/…)")
	frequency := fs.String("frequency", "", "intraday: check frequency (e.g. 5m)")
	tradeType := fs.String("trade-type", "", "intraday: signal type (e.g. 信号买入)")
	reportDate := fs.String("report-date", "", "intraday: YYYY-MM-DD (default today)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "usage: geegoo run <skill>")
		os.Exit(2)
	}
	skill := fs.Args()[0]

	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(2)
	}
	if application.Gateway == nil && !*dryRun {
		fmt.Fprintln(os.Stderr, "warning: LLM not configured (workflow stub mode)")
	}

	var runOpts app.SkillRunOptions
	if skill == "intraday" {
		in := workflow.DefaultIntradayInput()
		if v := strings.TrimSpace(*code); v != "" {
			in.Code = v
		}
		if v := strings.TrimSpace(*stockName); v != "" {
			in.StockName = v
		}
		if v := strings.TrimSpace(*botID); v != "" {
			in.BotID = v
		}
		if v := strings.TrimSpace(*botName); v != "" {
			in.BotName = v
		}
		if v := strings.TrimSpace(*botType); v != "" {
			in.BotType = v
		}
		if v := strings.TrimSpace(*frequency); v != "" {
			in.Frequency = v
		}
		if v := strings.TrimSpace(*tradeType); v != "" {
			in.TradeType = v
		}
		if v := strings.TrimSpace(*reportDate); v != "" {
			in.ReportDate = v
		}
		runOpts.Intraday = &in
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := application.RunSkillContext(ctx, skill, runOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("session=%s status=%s\n", result.SessionID, result.Status)
	if result.LastError != "" {
		fmt.Fprintf(os.Stderr, "error=%s\n", result.LastError)
	}
	if !result.OK() {
		os.Exit(1)
	}
}
