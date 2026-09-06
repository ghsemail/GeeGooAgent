//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func main() {
	var sqlite, pg strings.Builder
	sqlite.WriteString("DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';\n")
	sqlite.WriteString("DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';\n")
	pg.WriteString("DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';\n")
	pg.WriteString("DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';\n")

	for _, c := range eval.IndividualTurnPlanEvalCases() {
		stepsJSON, _ := json.Marshal(c.Steps)
		optsJSON, _ := json.Marshal(c.Options)
		title := escapeSQL(c.Title)
		desc := escapeSQL(c.Description)
		steps := escapeSQL(string(stepsJSON))
		opts := escapeSQL(string(optsJSON))

		fmt.Fprintf(&sqlite,
			"INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('%s', '', '%s', '%s', '%s', 0, '%s', %d, 1, datetime('now'), datetime('now'));\n",
			c.ID, title, desc, steps, opts, c.SortOrder,
		)
		fmt.Fprintf(&pg,
			"INSERT INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('%s', '', '%s', '%s', '%s', FALSE, '%s', %d, TRUE, NOW(), NOW()) ON CONFLICT (id) DO NOTHING;\n",
			c.ID, title, desc, steps, opts, c.SortOrder,
		)
	}

	if err := patchFile("internal/infra/schema.sql", "DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';", sqlite.String()); err != nil {
		panic(err)
	}
	if err := patchFile("internal/infra/pgschema/postgres_eval.sql", "DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';", pg.String()); err != nil {
		panic(err)
	}
	fmt.Println("patched schema.sql and postgres_eval.sql turn_plan seeds")
}

func patchFile(path, startMarker, replacement string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)
	idx := strings.Index(content, startMarker)
	if idx < 0 {
		return fmt.Errorf("%s: marker not found", path)
	}
	patched := content[:idx] + replacement
	return os.WriteFile(path, []byte(patched), 0o644)
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
