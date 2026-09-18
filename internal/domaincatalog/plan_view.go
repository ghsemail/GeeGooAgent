package domaincatalog

import (
	"fmt"
	"strings"
)

// PlanCatalogMarkdown renders domain → playbook → mode for dashboard / Memory → Plan.
func PlanCatalogMarkdown() string {
	var b strings.Builder
	b.WriteString("## Domain → Playbook → Mode\n\n")
	b.WriteString("| domain | mode | playbook | 说明 |\n")
	b.WriteString("|--------|------|----------|------|\n")
	for _, p := range plugins {
		skills := strings.Join(p.Skills, ", ")
		if skills == "" {
			skills = "—"
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n",
			p.Domain, p.Mode, skills, strings.TrimSpace(p.Reason))
	}
	b.WriteString("\n> 回测固定：`backtest_run` → **strategy-backtest-run** → `run_strategy_backtest`。\n")
	b.WriteString("> 生成策略：`dca_grid/execute` → **strategy-backtest** → `generate_*`（非普通回测）。\n")
	return b.String()
}
