package playbookexec

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunProbeSARMACDUsesAgentSellDefault(t *testing.T) {
	var probeArgs map[string]any
	router := &Router{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			switch req.Name {
			case "search_code":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{map[string]any{
						"code": "300308.SZ", "name": "中际旭创", "market": "CN",
					}}},
				}
			case "get_signal_combinations":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{map[string]any{
						"name":      "SAR信号配套MACD直方图趋势",
						"signal_id": slots.SARMACDCatalogSignalID,
						"buy_signal": []any{
							map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
							map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
						},
						"sell_signal": []any{
							map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}},
						},
						"frequency": "60m",
					}}},
				}
			case "probe_bot_signal_series":
				probeArgs = req.Arguments
				return tools.Result{Status: tools.StatusOK, Summary: "ok", Data: map[string]any{"buy_hits": 1, "sell_hits": 1}}
			default:
				return tools.Result{Status: tools.StatusError, Summary: req.Name}
			}
		},
	}
	result, ok := router.TryRunFromPlan(context.Background(), Input{
		Session:  runtime.NewSession(),
		UserText: "测一下中际旭创 SAR+MACD 买卖点",
		StepBase: 1,
	}, "signal_probe")
	if !ok || result.Failed {
		t.Fatalf("ok=%v failed=%v", ok, result.Failed)
	}
	if probeArgs == nil {
		t.Fatal("probe not called")
	}
	if _, ok := probeArgs["sell_merge"]; ok {
		t.Fatalf("sell_merge should be omitted: %v", probeArgs["sell_merge"])
	}
	sell, _ := probeArgs["sell_signal"].([]any)
	if len(sell) != 1 {
		t.Fatalf("sell=%v", sell)
	}
	row := sell[0].(map[string]any)
	if fmt.Sprint(row["index"]) != "MACD" || fmt.Sprint(row["type"]) != "signal" {
		t.Fatalf("sell rule=%v", row)
	}
	param, _ := row["param"].(map[string]any)
	if fmt.Sprint(param["fastPeriod"]) != "12" || fmt.Sprint(param["slowPeriod"]) != "26" || fmt.Sprint(param["signalPeriod"]) != "9" {
		t.Fatalf("param=%v", param)
	}
	buy, _ := probeArgs["buy_signal"].([]any)
	if len(buy) != 2 {
		t.Fatalf("buy=%v", buy)
	}
}
