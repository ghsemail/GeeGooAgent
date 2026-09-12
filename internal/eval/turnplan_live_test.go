package eval_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestIndividualTurnPlanEvalCasesCount(t *testing.T) {
	cases := eval.IndividualTurnPlanEvalCases()
	if len(cases) != 26 {
		t.Fatalf("cases=%d want 26", len(cases))
	}
	live := eval.DefaultTurnPlanLiveCases()
	if len(live) != 26 {
		t.Fatalf("live=%d want 26", len(live))
	}
	seen := map[string]int{}
	for _, c := range live {
		if c.Category == "" {
			t.Fatalf("%s missing category", c.ID)
		}
		seen[c.Category]++
	}
	for _, cat := range eval.TurnPlanCategories() {
		if seen[cat.ID] == 0 {
			t.Fatalf("category %s has no cases", cat.ID)
		}
	}
	for _, c := range cases {
		if c.Options.PlanOnly {
			t.Fatalf("%s still plan_only", c.ID)
		}
		if strings.TrimSpace(c.Options.Message) == "" {
			t.Fatalf("%s missing message", c.ID)
		}
	}
}

func TestPrintTurnPlanEvalSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("sql dump")
	}
	var b strings.Builder
	b.WriteString("DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';\n")
	b.WriteString("DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';\n")
	for _, c := range eval.IndividualTurnPlanEvalCases() {
		stepsJSON, _ := json.Marshal(c.Steps)
		optsJSON, _ := json.Marshal(c.Options)
		fmt.Fprintf(&b, "INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('%s', '', '%s', '%s', '%s', 0, '%s', %d, 1, datetime('now'), datetime('now'));\n",
			c.ID,
			escapeSQL(c.Title),
			escapeSQL(c.Description),
			escapeSQL(string(stepsJSON)),
			escapeSQL(string(optsJSON)),
			c.SortOrder,
		)
	}
	t.Log("\n" + b.String())
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
