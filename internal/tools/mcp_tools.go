package tools

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

// MCPDeps bundles MCP client for tool handlers.
type MCPDeps struct {
	Client *mcp.Client
}

// RegisterChatMCPTools wires A2 chat tools to GeeGooBot mcp-api (:3120).
func RegisterChatMCPTools(r *Registry, deps MCPDeps) {
	r.Register(Tool{
		Name:        "search_code",
		Description: "Search stock by code or name via GeeGooBot MCP /searchCode (no mcp_token).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"regex":  map[string]any{"type": "string", "description": "Code or name regex"},
				"market": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"regex"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			regex, _ := args["regex"].(string)
			markets := stringSliceArg(args["market"])
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: "dry-run: skipped search_code", Data: map[string]any{"items": []any{}}}
			}
			items, err := deps.Client.SearchCode(context.Background(), regex, markets)
			if err != nil {
				return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1}
			}
			out := make([]map[string]string, 0, len(items))
			for _, it := range items {
				out = append(out, map[string]string{"code": it.Code, "name": it.Name})
			}
			return Result{
				Status:  StatusOK,
				Summary: fmt.Sprintf("search_code: %d item(s)", len(out)),
				Data:    map[string]any{"items": out},
			}
		},
	})

	r.Register(Tool{
		Name:        "get_current_price",
		Description: "Get latest price via GeeGooBot MCP /getCurrentPrice.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "Ticker e.g. 00700.HK"},
			},
			"required": []string{"code"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code, _ := args["code"].(string)
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: "dry-run: skipped get_current_price", Data: map[string]any{"code": code}}
			}
			price, err := deps.Client.GetCurrentPrice(context.Background(), ctx.MCPToken, code)
			if err != nil {
				return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1}
			}
			return Result{
				Status:  StatusOK,
				Summary: fmt.Sprintf("%s price=%v", code, price.Price),
				Data:    map[string]any{"code": code, "price": price.Price, "source": "GeeGooBot-mcp-api"},
			}
		},
	})

	r.Register(Tool{
		Name:        "check_trading_day",
		Description: "Check if today is a trading day for the market of the given code.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "Stock code e.g. 00700.HK"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code, _ := args["code"].(string)
			if code == "" {
				code = "00700.HK"
			}
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: "dry-run: skipped check_trading_day", Data: map[string]any{"is_trading_day": true}}
			}
			data, err := deps.Client.CheckTradingDay(context.Background(), ctx.MCPToken, code)
			if err != nil {
				return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1}
			}
			return Result{
				Status: StatusOK,
				Summary: fmt.Sprintf(
					"Trading day check: %s on %s market=%s is_trading_day=%v",
					data.Code, data.Date, data.Market, data.IsTradingDay,
				),
				Data: map[string]any{
					"is_trading_day": data.IsTradingDay,
					"date":           data.Date,
					"market":         data.Market,
					"code":           data.Code,
				},
			}
		},
	})
}

func stringSliceArg(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
