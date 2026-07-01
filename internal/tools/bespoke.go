package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/search"
)

const (
	indexPromptID     = "69ec7035b9ccd3d9befc6c23"
	aShareCapitalSkip = "A-share capital flow not available for this account"
)

var dryRunSampleBots = []map[string]string{
	{
		"stock_name": "腾讯控股", "code": "00700.HK", "bot_id": "dry-run-bot-1",
		"bot_name": "dry-run-bot", "bot_type": "DCA",
	},
}

// RegisterBespokeTools registers hand-written MCP and local tools.
func RegisterBespokeTools(r *Registry, deps Deps) {
	registerPerceptionTools(r, deps)
	registerAnalysisTools(r, deps)
	registerReportTools(r, deps)
	registerMetaTools(r, deps)
}

func registerPerceptionTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name:        "search_code",
		Description: "在 GeeGoo 股票库中按代码或名称（中/英/繁）模糊搜索，含 SpaceX 等特殊标的。查价/分析/找 Bot 标的时优先使用。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"regex": map[string]any{
					"type":        "string",
					"description": "代码或名称关键词，如 spacex、00700、腾讯",
				},
				"market": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "可选市场过滤，如 [\"HK\",\"US\"]",
				},
			},
			"required": []any{"regex"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			regex := strArg(args, "regex", "")
			markets := stringSliceArg(args["market"])
			if ctx.DryRun {
				return okDryRun("search_code", map[string]any{"items": []any{}})
			}
			items, err := deps.MCP.SearchCode(context.Background(), regex, markets)
			if err != nil {
				return errResult(err)
			}
			anyItems := make([]any, 0, len(items))
			for _, it := range items {
				row := map[string]any{"code": it.Code, "name": it.Name}
				if it.NameEN != "" {
					row["name_en"] = it.NameEN
				}
				if it.NameZH != "" {
					row["name_zh"] = it.NameZH
				}
				if it.Market != "" {
					row["market"] = it.Market
				}
				if it.StockType != "" {
					row["stock_type"] = it.StockType
				}
				if it.LotSize > 0 {
					row["lot_size"] = it.LotSize
				}
				anyItems = append(anyItems, row)
			}
			summary := fmt.Sprintf("search_code: %d item(s)", len(items))
			if len(items) > 0 {
				top := items[0]
				label := top.Name
				if label == "" {
					label = top.NameEN
				}
				summary = fmt.Sprintf("search_code: %d item(s); top: %s (%s)", len(items), label, top.Code)
			}
			return Result{Status: StatusOK, Summary: summary, Data: map[string]any{"items": anyItems}}
		},
	})
	r.Register(Tool{
		Name:        "web_search",
		Description: "网页搜索（免费 DuckDuckGo）。仅当 search_code 在 GeeGoo 股票库无结果、且需要外部新闻/时事时使用。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query in Chinese or English",
				},
			},
			"required": []any{"query"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			query := strArg(args, "query", "")
			if query == "" {
				return Result{Status: StatusError, Summary: "web_search: query required", ExitCode: 1}
			}
			if ctx.DryRun {
				return okDryRun("web_search", map[string]any{"query": query, "results": []any{}})
			}
			cfg := deps.Search
			if strings.TrimSpace(cfg.Provider) == "" {
				cfg.Provider = search.ProviderDuckDuckGo
			}
			if cfg.MaxResults <= 0 {
				cfg.MaxResults = 5
			}
			hits, err := search.Search(context.Background(), search.Config{
				Provider: cfg.Provider, MaxResults: cfg.MaxResults,
			}, query)
			if err != nil {
				return errResult(err)
			}
			if len(hits) == 0 {
				return Result{Status: StatusOK, Summary: "web_search: no results",
					Data: map[string]any{"query": query, "results": []any{}, "count": 0}}
			}
			items := make([]map[string]any, 0, len(hits))
			for _, h := range hits {
				items = append(items, map[string]any{
					"title": h.Title, "url": h.URL, "snippet": h.Snippet,
				})
			}
			summary := fmt.Sprintf("web_search: %d hit(s); top: %s", len(items), shorten(hits[0].Title, 80))
			return Result{Status: StatusOK, Summary: summary,
				Data: map[string]any{"query": query, "count": len(items), "results": items}}
		},
	})
	r.Register(Tool{
		Name: "check_trading_day", Description: "Check if today is a trading day.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "00700.HK")
			if ctx.DryRun {
				return okDryRun("check_trading_day", map[string]any{"is_trading_day": true, "code": code, "market": "HK", "date": today()})
			}
			data, err := deps.MCP.CheckTradingDay(context.Background(), ctx.MCPToken, code)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("is_trading_day=%v", data.IsTradingDay), Data: map[string]any{
				"is_trading_day": data.IsTradingDay, "date": data.Date, "market": data.Market, "code": data.Code,
			}}
		},
	})
	r.Register(Tool{
		Name: "get_current_price", Description: "Get latest price via GeeGooBot MCP.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return okDryRun("get_current_price", map[string]any{"code": code, "price": nil})
			}
			price, err := deps.MCP.GetCurrentPrice(context.Background(), ctx.MCPToken, code)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("%s price=%v", code, price.Price),
				Data: map[string]any{"code": code, "price": price.Price, "source": "GeeGooBot-mcp-api"}}
		},
	})
	r.Register(Tool{
		Name: "get_report_bot_codes", Description: "List stocks/bots for report workflow.",
		Handle: func(ctx Context, args map[string]any) Result {
			_ = args
			if ctx.DryRun {
				items := make([]map[string]string, len(dryRunSampleBots))
				copy(items, dryRunSampleBots)
				return Result{Status: StatusDryRun, Summary: fmt.Sprintf("dry-run: %d sample bot(s)", len(items)),
					Data: map[string]any{"bots": toAnySlice(items)}}
			}
			bots, err := deps.MCP.GetReportBotCodes(context.Background(), ctx.MCPToken)
			if err != nil {
				return errResult(err)
			}
			out := make([]map[string]any, 0, len(bots))
			for _, b := range bots {
				out = append(out, map[string]any{
					"code": b.Code, "stock_name": b.StockName, "bot_id": b.BotID,
					"bot_name": b.BotName, "bot_type": b.BotType,
				})
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Found %d bot(s)", len(out)), Data: map[string]any{"bots": out}}
		},
	})
	r.Register(Tool{
		Name: "fetch_market_news", Description: "Fetch market news (bundled scripts when not dry-run).",
		Handle: func(ctx Context, args map[string]any) Result {
			market := strArg(args, "market", "US")
			if ctx.DryRun {
				return okDryRun("fetch_market_news", map[string]any{"market": market, "text": "", "items": []any{}})
			}
			return Result{
				Status:  StatusSkip,
				Summary: "fetch_market_news: script runner unavailable; continuing without bundled news",
				Data:    map[string]any{"market": market, "text": "", "items": []any{}, "source": "unavailable"},
			}
		},
	})
	r.Register(Tool{
		Name: "fetch_stock_news", Description: "Fetch stock-specific news.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return okDryRun("fetch_stock_news", map[string]any{"code": code, "text": "", "source": "dry-run"})
			}
			return Result{
				Status:  StatusSkip,
				Summary: "fetch_stock_news: script runner unavailable; continuing without bundled news",
				Data:    map[string]any{"code": code, "text": "", "source": "unavailable"},
			}
		},
	})
}

func registerAnalysisTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "get_mcp_analysis", Description: "Run MCP technical analysis.",
		Handle: func(ctx Context, args map[string]any) Result {
			name := strArg(args, "name", "")
			code := strArg(args, "code", "")
			period := strArg(args, "period", "hourly")
			promptID := strArg(args, "prompt_id", indexPromptID)
			if ctx.DryRun {
				return okDryRun("get_mcp_analysis", map[string]any{
					"code": code, "period": period, "analysis_result": fmt.Sprintf("[dry-run] analysis for %s", name),
				})
			}
			result, err := deps.MCP.GetMCPAnalysis(context.Background(), ctx.MCPToken, name, code, promptID, period, "cn")
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("MCP analysis %s (%s)", code, period), Data: map[string]any{
				"code": code, "period": period, "analysis_result": result.AnalysisResult,
				"model": result.Model, "create_date": result.CreateDate,
			}}
		},
	})
	r.Register(Tool{
		Name: "get_capital_flow", Description: "Fetch capital flow for a stock.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			period := strArg(args, "period", "DAY")
			if isAShare(code) {
				return Result{Status: StatusSkip, Summary: aShareCapitalSkip, Data: map[string]any{
					"code": code, "skip_reason": aShareCapitalSkip, "items": []any{}, "latest": map[string]any{},
				}}
			}
			if ctx.DryRun {
				return okDryRun("get_capital_flow", map[string]any{"code": code, "items": []any{}})
			}
			flows, err := deps.MCP.GetCapitalFlow(context.Background(), ctx.MCPToken, code, period, "")
			if err != nil {
				return errResult(err)
			}
			latest := map[string]any{}
			if len(flows) > 0 {
				latest = map[string]any{"main_in_flow": flows[len(flows)-1].MainInFlow}
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("capital flow %s", code),
				Data: map[string]any{"code": code, "latest": latest}}
		},
	})
	r.Register(Tool{
		Name: "get_capital_distribution", Description: "Fetch capital distribution.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if isAShare(code) {
				return Result{Status: StatusSkip, Summary: aShareCapitalSkip, Data: map[string]any{"code": code}}
			}
			if ctx.DryRun {
				return okDryRun("get_capital_distribution", map[string]any{"code": code})
			}
			dist, err := deps.MCP.GetCapitalDistribution(context.Background(), ctx.MCPToken, code)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("capital distribution %s", code), Data: map[string]any{
				"code": code, "formatted": fmt.Sprintf("super_in=%v", dist.CapitalInSuper),
			}}
		},
	})
	r.Register(Tool{
		Name: "get_bot_yesterday_attitude", Description: "Get bot yesterday attitude.",
		Handle: func(ctx Context, args map[string]any) Result {
			botID := strArg(args, "bot_id", "")
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return okDryRun("get_bot_yesterday_attitude", map[string]any{"bot_id": botID, "code": code, "attitude": "neutral"})
			}
			att, err := deps.MCP.GetBotYesterdayAttitude(context.Background(), ctx.MCPToken, botID, "cn")
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("attitude=%s", att.Attitude), Data: map[string]any{
				"bot_id": botID, "code": code, "attitude": att.Attitude, "found": att.Found,
			}}
		},
	})
	r.Register(Tool{
		Name: "get_stock_daily_reports", Description: "Query aggregated daily reports.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			reportDate := strArg(args, "report_date", today())
			if ctx.DryRun {
				return okDryRun("get_stock_daily_reports", map[string]any{"pre_market": []any{}, "intraday": []any{}, "post_market": []any{}})
			}
			reports, err := deps.MCP.GetStockDailyReports(context.Background(), ctx.MCPToken, code, reportDate)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("daily reports %s", code), Data: map[string]any{
				"pre_market": reports.PreMarket, "intraday": reports.Intraday, "post_market": reports.PostMarket,
			}}
		},
	})
	r.Register(Tool{
		Name: "list_today_reports", Description: "Idempotency check for today's pre_market reports.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			reportDate := strArg(args, "report_date", today())
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: fmt.Sprintf("dry-run: no existing reports for %s", code),
					Data: map[string]any{"code": code, "report_date": reportDate, "count": 0, "already_reported": false}}
			}
			reports, err := deps.MCP.GetStockDailyReports(context.Background(), ctx.MCPToken, code, reportDate)
			if err != nil {
				return errResult(err)
			}
			count := len(reports.PreMarket)
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Found %d pre_market report(s)", count), Data: map[string]any{
				"code": code, "report_date": reportDate, "count": count,
				"reports": reports.PreMarket, "already_reported": count > 0,
			}}
		},
	})
	r.Register(Tool{
		Name: "recall_yesterday_summary", Description: "Recall yesterday summary (stub).",
		Handle: func(ctx Context, args map[string]any) Result {
			_ = ctx
			_ = args
			return Result{Status: StatusOK, Summary: "recall_yesterday_summary: not implemented", Data: map[string]any{}}
		},
	})
	r.Register(Tool{
		Name: "read_working_state", Description: "Read workflow working memory.",
		Handle: func(ctx Context, args map[string]any) Result {
			_ = args
			if deps.Working == nil {
				return Result{Status: StatusError, Summary: "working store not configured", ExitCode: 1}
			}
			data, err := deps.Working.Load(ctx.SessionID)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: "working state loaded", Data: data}
		},
	})
	r.Register(Tool{
		Name: "recall",
		Description: "Search past geegoo chat sessions for stock price lookups and user queries. " +
			"Use when the user asks what they checked before, including after quit/restart.",
		Handle: func(ctx Context, args map[string]any) Result {
			if ctx.StateStore == nil {
				return Result{Status: StatusError, Summary: "state_store not configured", ExitCode: 1}
			}
			query := strArg(args, "query", "")
			limit := intArg(args, "limit", 5)
			if limit < 1 {
				limit = 1
			}
			if limit > 20 {
				limit = 20
			}
			store := chatsession.NewChatSessionStore(ctx.StateStore)
			hits, err := chatsession.SearchPastSessions(store, query, ctx.SessionID, limit, 30)
			if err != nil {
				return errResult(err)
			}
			if len(hits) == 0 {
				return Result{
					Status: StatusOK, Summary: "No matching past chat sessions",
					Data: map[string]any{"count": 0, "matches": []any{}},
				}
			}
			top := hits[0]
			summary := fmt.Sprintf("Found %d session(s); latest: %s (%s)", len(hits), top.Snippet, top.SessionID)
			for _, e := range top.StockEvents {
				if e.Code != "" && e.Tool == "get_current_price" {
					priceNote := ""
					if e.Price != nil {
						priceNote = fmt.Sprintf(" @ %v", *e.Price)
					}
					summary = fmt.Sprintf("Found %d session(s); latest: %s%s (%s)", len(hits), e.Code, priceNote, top.SessionID)
					break
				}
			}
			return Result{Status: StatusOK, Summary: summary, Data: chatsession.HitsToData(hits)}
		},
	})
}

func registerReportTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "create_pre_market_report", Description: "Create pre-market report via GeeGooBot MCP.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: fmt.Sprintf("dry-run: skipped create_pre_market_report %s", code),
					Data: map[string]any{"report_id": "dry-run-id", "code": code}}
			}
			result, err := deps.MCP.CreatePreMarketReport(context.Background(), ctx.MCPToken, args)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Created report for %s", code),
				Data: map[string]any{"report_id": result.ReportID, "code": code}}
		},
	})
	r.Register(Tool{
		Name: "save_local_report", Description: "Save report markdown under workspace.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			content := strArg(args, "content", "")
			reportType := strArg(args, "report_type", "premarket")
			reportDate := strArg(args, "report_date", today())
			suffix := reportType
			if suffix == "premarket" {
				suffix = "premarket"
			}
			guard, err := infra.NewWorkspaceGuard(deps.WorkspaceRoot)
			if err != nil {
				return errResult(err)
			}
			rel := fmt.Sprintf("reports/%s/%s-%s.md", reportDate, code, suffix)
			path, err := guard.Resolve(rel)
			if err != nil {
				return errResult(err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return errResult(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Saved %s", filepath.Base(path)),
				Data: map[string]any{"path": path, "code": code}}
		},
	})
	r.Register(Tool{
		Name: "send_feishu_summary", Description: "Send Feishu webhook summary (stub).",
		Handle: func(ctx Context, args map[string]any) Result {
			_ = args
			if ctx.DryRun {
				return okDryRun("send_feishu_summary", map[string]any{})
			}
			return Result{Status: StatusSkip, Summary: "feishu webhook not configured"}
		},
	})
}

func registerMetaTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "write_execution_log", Description: "Append workflow step to execution log.",
		Handle: func(ctx Context, args map[string]any) Result {
			step := strArg(args, "step", "")
			message := strArg(args, "message", "")
			status := strArg(args, "status", "ok")
			guard, err := infra.NewWorkspaceGuard(deps.WorkspaceRoot)
			if err != nil {
				return errResult(err)
			}
			rel := fmt.Sprintf("%s/execution-log.md", today())
			path, err := guard.Resolve(rel)
			if err != nil {
				return errResult(err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return errResult(err)
			}
			existing := ""
			if raw, err := os.ReadFile(path); err == nil {
				existing = string(raw)
			} else {
				existing = fmt.Sprintf("# Execution Log — %s\n\n", ctx.SessionID)
			}
			line := fmt.Sprintf("- [%s] %s: %s\n", status, step, message)
			if err := os.WriteFile(path, []byte(existing+line), 0o644); err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Logged %s", step), Data: map[string]any{"path": path}}
		},
	})
	_ = deps
}

func strArg(args map[string]any, key, def string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return def
}

func intArg(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return def
	}
}

func today() string { return time.Now().Format("2006-01-02") }

func isAShare(code string) bool {
	return strings.HasSuffix(code, ".SH") || strings.HasSuffix(code, ".SZ")
}

func okDryRun(name string, data map[string]any) Result {
	return Result{Status: StatusDryRun, Summary: "dry-run: skipped " + name, Data: data}
}

func errResult(err error) Result {
	return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1}
}

func toAnySlice(in []map[string]string) []any {
	out := make([]any, len(in))
	for i, m := range in {
		row := map[string]any{}
		for k, v := range m {
			row[k] = v
		}
		out[i] = row
	}
	return out
}

func shorten(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
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
