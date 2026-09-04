package playbookexec

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Router) resolveStock(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (code, name, market string, err error) {
	runTool := func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		return r.runTool(ctx, tc, req.Name, req.Arguments, recordTool)
	}
	return slots.ResolveStock(ctx, toolCtx, runTool, plan.StockQuery)
}

func (r *Router) resolveSignals(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (buy, sell []any, frequency, strategyLabel string, err error) {
	runTool := func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		return r.runTool(ctx, tc, req.Name, req.Arguments, recordTool)
	}
	sig, err := slots.ResolveSignal(ctx, toolCtx, runTool, slots.SignalPlan{
		SignalQuery: plan.SignalQuery,
		SignalKind:  plan.SignalKind,
	})
	if err != nil {
		return nil, nil, "", "", err
	}
	return sig.Buy, sig.Sell, sig.Frequency, sig.StrategyLabel, nil
}

func catalogItems(data map[string]any) []map[string]any {
	return slots.CatalogItems(data)
}

func formatBacktestReply(code, name, strategyLabel string, data map[string]any) string {
	profit := fmt.Sprint(data["profit_rate"])
	finalValue := fmt.Sprint(data["final_value"])
	logID := fmt.Sprint(data["log_id"])
	trades := fmt.Sprint(data["trade_count"])
	drawdown := fmt.Sprint(data["drawdown"])
	return fmt.Sprintf("## %s %s · SmartTrade 回测\n\n- **策略**：%s\n- **收益率**：%s%%\n- **最终资产**：%s\n- **最大回撤**：%s%%\n- **成交笔数**：%s\n- **log_id**：%s\n\n> 由 playbook 确定性执行（run_strategy_backtest）",
		name, code, strategyLabel, profit, finalValue, drawdown, trades, logID)
}
