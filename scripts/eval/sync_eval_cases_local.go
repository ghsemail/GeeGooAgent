//go:build ignore

// sync_eval_cases_local upserts TurnPlan + strategy_backtest eval rows into local SQLite.
// Usage: go run scripts/eval/sync_eval_cases_local.go [-db path/to/geegoo.db]
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func main() {
	dbPath := flag.String("db", "", "SQLite path (default ~/.geegoo/data/geegoo.db)")
	flag.Parse()

	path := strings.TrimSpace(*dbPath)
	if path == "" {
		path = filepath.Join(config.Home(), "data", "geegoo.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}

	db, err := infra.OpenSQLite(path)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlDB := db.SQL()
	nTurn, err := syncTurnPlanCases(sqlDB)
	if err != nil {
		panic(err)
	}
	nBacktest, err := syncStrategyBacktestCases(sqlDB)
	if err != nil {
		panic(err)
	}
	nWorkflow, err := syncWorkflowCases(sqlDB)
	if err != nil {
		panic(err)
	}
	fmt.Printf("synced %d turn_plan + %d strategy_backtest + %d workflow cases into %s\n", nTurn, nBacktest, nWorkflow, path)
}

func syncWorkflowCases(db *sql.DB) (int, error) {
	if _, err := db.Exec(`DELETE FROM agent_eval_cases WHERE id IN ('taskflow_multi_strategy_compare', 'workflow_strategy_dev_cognition')`); err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	cases := eval.IndividualWorkflowEvalCases()
	for _, c := range cases {
		stepsJSON, err := json.Marshal(c.Steps)
		if err != nil {
			return 0, err
		}
		optsJSON, err := json.Marshal(c.Options)
		if err != nil {
			return 0, err
		}
		_, err = db.Exec(`
			INSERT OR REPLACE INTO agent_eval_cases (
				id, user_id, title, description, steps_json, supports_random_stock,
				options_json, sort_order, enabled, created_at, updated_at
			) VALUES (?, '', ?, ?, ?, 0, ?, ?, 1, ?, ?)`,
			c.ID, c.Title, c.Description, string(stepsJSON), string(optsJSON), c.SortOrder, now, now,
		)
		if err != nil {
			return 0, fmt.Errorf("upsert %s: %w", c.ID, err)
		}
	}
	return len(cases), nil
}

func syncTurnPlanCases(db *sql.DB) (int, error) {
	if _, err := db.Exec(`DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%'`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing'`); err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	cases := eval.IndividualTurnPlanEvalCases()
	for _, c := range cases {
		stepsJSON, err := json.Marshal(c.Steps)
		if err != nil {
			return 0, err
		}
		optsJSON, err := json.Marshal(c.Options)
		if err != nil {
			return 0, err
		}
		_, err = db.Exec(`
			INSERT OR REPLACE INTO agent_eval_cases (
				id, user_id, title, description, steps_json, supports_random_stock,
				options_json, sort_order, enabled, created_at, updated_at
			) VALUES (?, '', ?, ?, ?, 0, ?, ?, 1, ?, ?)`,
			c.ID, c.Title, c.Description, string(stepsJSON), string(optsJSON), c.SortOrder, now, now,
		)
		if err != nil {
			return 0, fmt.Errorf("upsert %s: %w", c.ID, err)
		}
	}
	return len(cases), nil
}

func syncStrategyBacktestCases(db *sql.DB) (int, error) {
	type row struct {
		steps []string
		opts  map[string]any
	}
	cases := map[string]row{
		"strategy_backtest_single": {
			steps: []string{
				"随机选股与策略",
				"发送回测请求",
				"若 Agent clarify：由 auto-clarify 自动应答",
				"校验回复含收益率与 log_id",
			},
			opts: map[string]any{
				"category": "strategy_backtest", "task": "backtest", "scenario": "single",
				"stock_count": 1, "strategy_count": 1, "random_stock_enabled": true,
				"min_reply_chars": 80, "pass_keywords": []string{"回测", "收益", "log"},
				"session_cleanup": "before_run", "auto_clarify_only": true,
			},
		},
		"strategy_backtest_multi_strategy": {
			steps: []string{
				"随机选股与 2 项策略",
				"发送多策略回测对比请求",
				"若 Agent clarify：由 auto-clarify 自动应答",
				"校验回复含收益对比与 log_id",
			},
			opts: map[string]any{
				"category": "strategy_backtest", "task": "backtest", "scenario": "multi_strategy",
				"stock_count": 1, "strategy_count": 2, "random_stock_enabled": true,
				"min_reply_chars": 100, "pass_keywords": []string{"回测", "对比", "log"},
				"session_cleanup": "before_run", "auto_clarify_only": true,
			},
		},
		"strategy_backtest_multi_config": {
			steps: []string{
				"随机选股与策略",
				"发送多止盈止损参数回测请求",
				"若 Agent clarify：由 auto-clarify 自动应答",
				"校验回复含各套配置收益对比",
			},
			opts: map[string]any{
				"category": "strategy_backtest", "task": "backtest", "scenario": "multi_config",
				"stock_count": 1, "strategy_count": 1, "random_stock_enabled": true,
				"config_variants": []string{"止盈5%止损3%", "止盈7%止损5%"},
				"min_reply_chars": 120, "pass_keywords": []string{"回测", "对比", "止盈"},
				"session_cleanup": "before_run", "auto_clarify_only": true,
			},
		},
	}
	now := time.Now().UTC().Format(time.RFC3339)
	n := 0
	for id, c := range cases {
		stepsJSON, err := json.Marshal(c.steps)
		if err != nil {
			return n, err
		}
		optsJSON, err := json.Marshal(c.opts)
		if err != nil {
			return n, err
		}
		res, err := db.Exec(`
			UPDATE agent_eval_cases
			SET steps_json = ?, options_json = ?, updated_at = ?
			WHERE id = ?`, string(stepsJSON), string(optsJSON), now, id)
		if err != nil {
			return n, err
		}
		aff, _ := res.RowsAffected()
		if aff == 0 {
			_, err = db.Exec(`
				INSERT INTO agent_eval_cases (
					id, user_id, title, description, steps_json, supports_random_stock,
					options_json, sort_order, enabled, created_at, updated_at
				) VALUES (?, '', ?, ?, ?, 1, ?, ?, 1, ?, ?)`,
				id, titleForBacktest(id), descForBacktest(id), string(stepsJSON), string(optsJSON), sortForBacktest(id), now, now,
			)
			if err != nil {
				return n, fmt.Errorf("insert %s: %w", id, err)
			}
		}
		n++
	}
	return n, nil
}

func titleForBacktest(id string) string {
	switch id {
	case "strategy_backtest_single":
		return "单股 · 单策略 · 回测"
	case "strategy_backtest_multi_strategy":
		return "单股 · 多策略 · 回测"
	case "strategy_backtest_multi_config":
		return "单股 · 单策略 · 多止盈止损回测"
	default:
		return id
	}
}

func descForBacktest(id string) string {
	switch id {
	case "strategy_backtest_single":
		return "随机单股单策略跑 run_strategy_backtest，汇报收益与 log_id。"
	case "strategy_backtest_multi_strategy":
		return "同一只股票上对比 2 项随机策略的回测收益。"
	case "strategy_backtest_multi_config":
		return "同一策略用两套止盈止损参数（如 5%/3% vs 7%/5%）跑回测并对比。"
	default:
		return ""
	}
}

func sortForBacktest(id string) int {
	switch id {
	case "strategy_backtest_single":
		return 20
	case "strategy_backtest_multi_strategy":
		return 21
	case "strategy_backtest_multi_config":
		return 22
	default:
		return 0
	}
}
