package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// notifyStock replays Feishu digest for a completed stock workflow session.
func notifyStock(args []string) {
	fs := flag.NewFlagSet("notify-stock", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	sessionID := fs.String("session", "", "workflow session id (run-...)")
	skill := fs.String("skill", "", "premarket_stock or postmarket_stock")
	market := fs.String("market", "", "CN, HK, or US")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	sid := strings.TrimSpace(*sessionID)
	skillName := strings.TrimSpace(*skill)
	mkt := strings.ToUpper(strings.TrimSpace(*market))
	if sid == "" || skillName == "" || mkt == "" {
		fmt.Fprintln(os.Stderr, "usage: geegoo notify-stock --session ID --skill premarket_stock|postmarket_stock --market CN|HK|US")
		os.Exit(2)
	}
	if !defaultFeishuNotifyForSkill(skillName) {
		fmt.Fprintf(os.Stderr, "notify-stock: unsupported skill %q\n", skillName)
		os.Exit(2)
	}

	application, err := app.LoadFromConfigPath(*configPath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "notify-stock: %v\n", err)
		os.Exit(2)
	}
	if application.Working == nil {
		fmt.Fprintln(os.Stderr, "notify-stock: working store not configured")
		os.Exit(2)
	}
	working, err := application.Working.Load(sid)
	if err != nil || working == nil {
		fmt.Fprintf(os.Stderr, "notify-stock: load working %s: %v\n", sid, err)
		os.Exit(2)
	}
	result := workflow.RunResult{
		SessionID: sid,
		Status:    "completed",
		Working:   working,
		Supervisor: &workflow.SupervisorReport{
			Verdict: workflow.VerdictPass,
		},
	}
	ctx := context.Background()
	sent, err := application.NotifyStockFeishuForUsers(ctx, skillName, mkt, result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "notify-stock: %v\n", err)
		os.Exit(2)
	}
	if sent == 0 {
		fmt.Fprintln(os.Stderr, "notify-stock: no digest sent (check logs for skip reason)")
		os.Exit(1)
	}
	fmt.Printf("notify-stock: sent=%d session=%s\n", sent, sid)
}
