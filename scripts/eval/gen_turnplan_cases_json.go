//go:build ignore

// Usage (repo root): go run scripts/eval/gen_turnplan_cases_json.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

type manifestCase struct {
	ID           string   `json:"id"`
	TurnID       string   `json:"turn_id"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Setup        []string `json:"setup"`
	Message      string   `json:"message"`
	ExpectDomain string   `json:"expect_domain"`
	ExpectMode   string   `json:"expect_mode"`
	ExpectSOP    bool     `json:"expect_sop"`
	RequireTools []string `json:"require_tools,omitempty"`
	ForbidTools  []string `json:"forbid_tools,omitempty"`
}

type manifest struct {
	Version    int                      `json:"version"`
	Categories []eval.TurnPlanCategory  `json:"categories"`
	Cases      []manifestCase           `json:"cases"`
}

func main() {
	live := eval.DefaultTurnPlanLiveCases()
	byTurn := make(map[string]eval.TurnPlanLiveCase, len(live))
	for _, c := range live {
		byTurn[c.ID] = c
	}

	out := manifest{Version: 1, Categories: eval.TurnPlanCategories()}
	for _, def := range eval.IndividualTurnPlanEvalCases() {
		src, ok := byTurn[def.Options.TurnID]
		if !ok {
			panic("missing live case for " + def.ID)
		}
		out.Cases = append(out.Cases, manifestCase{
			ID:           def.ID,
			TurnID:       def.Options.TurnID,
			Category:     src.Category,
			Title:        def.Title,
			Description:  def.Description,
			Setup:        append([]string(nil), def.Options.SetupMessages...),
			Message:      def.Options.Message,
			ExpectDomain: def.Options.ExpectDomain,
			ExpectMode:   def.Options.ExpectMode,
			ExpectSOP:    def.Options.ExpectSOP,
			RequireTools: append([]string(nil), def.Options.RequireTools...),
			ForbidTools:  append([]string(nil), def.Options.ForbidTools...),
		})
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	path := filepath.Join("scripts", "eval", "turnplan_cases.json")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		panic(err)
	}
	println("wrote", path, "cases=", len(out.Cases))
}
