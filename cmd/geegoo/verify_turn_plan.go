package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func runVerifyTurnPlan(args []string) {
	fs := flag.NewFlagSet("verify turn-plan", flag.ExitOnError)
	casesFile := fs.String("cases", "", "optional JSON file with TurnPlanSuite (default: built-in suite)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	suite := eval.DefaultTurnPlanSuite()
	if *casesFile != "" {
		raw, err := os.ReadFile(*casesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "verify turn-plan: %v\n", err)
			os.Exit(2)
		}
		if err := json.Unmarshal(raw, &suite); err != nil {
			fmt.Fprintf(os.Stderr, "verify turn-plan: parse cases: %v\n", err)
			os.Exit(2)
		}
	}

	results := eval.RunTurnPlanSuite(suite)
	for _, r := range results {
		mark := "✓"
		if !r.Passed {
			mark = "✗"
		}
		fmt.Printf("  %s %-28s %s\n", mark, r.TurnID, r.Detail)
	}
	if !eval.AllTurnPlanPass(results) {
		os.Exit(1)
	}
	fmt.Println("\nTurnPlan eval: PASS")
}
