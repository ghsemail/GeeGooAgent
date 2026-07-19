package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/search"
	"github.com/ghsemail/GeeGooAgent/internal/tools/newsrunner"
)

const indexPromptID = "69ec7035b9ccd3d9befc6c23"

var dryRunSampleBots = []map[string]string{
	{
		"stock_name": "腾讯控股", "code": "00700.HK", "bot_id": "dry-run-bot-1",
		"bot_name": "dry-run-bot", "bot_type": "DCA",
	},
}

// RegisterBespokeTools registers hand-written MCP and local tools.
func RegisterBespokeTools(r *Registry, deps Deps) {
	registerClarifyTool(r)
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
			items, err := deps.HTTP.SignalAPI.SearchCode(ctx.GoContext(), regex, markets)
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
			hits, err := search.Search(ctx.GoContext(), search.Config{
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
			data, err := deps.HTTP.MCP.CheckTradingDay(ctx.GoContext(), ctx.MCPToken, code)
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("is_trading_day=%v", data.IsTradingDay), Data: map[string]any{
				"is_trading_day": data.IsTradingDay, "date": data.Date, "market": data.Market, "code": data.Code,
			}}
		},
	})
	r.Register(Tool{
		Name:        "get_current_price",
		Description: "获取股票最新现价。须先 search_code 得到 code（如 00700.HK），再传入 code。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{
					"type":        "string",
					"description": "股票代码，如 00700.HK、AAPL.US",
				},
			},
			"required": []any{"code"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := firstStringArg(args, "code", "symbol", "stock_code", "ticker")
			if code == "" {
				return Result{Status: StatusError, Summary: "get_current_price 需要参数 code（如 00700.HK）", ExitCode: 1}
			}
			if ctx.DryRun {
				return okDryRun("get_current_price", map[string]any{"code": code, "price": nil})
			}
			price, err := deps.HTTP.MCP.GetCurrentPrice(ctx.GoContext(), ctx.MCPToken, code)
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
			bots, err := deps.HTTP.MCP.GetReportBotCodes(ctx.GoContext(), ctx.MCPToken)
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
		Name: "fetch_market_news", Description: "拉取市场新闻（经 GeeGooBot→GeeGooData 多源聚合；失败时本地 finance-news / web_search）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"market": map[string]any{"type": "string", "description": "US/CN/HK，默认 US"},
				"limit":  map[string]any{"type": "integer", "description": "条数，默认 8"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			market := strArg(args, "market", "US")
			limit := intArg(args, "limit", 8)
			if ctx.DryRun {
				return okDryRun("fetch_market_news", map[string]any{"market": market, "text": "", "items": []any{}})
			}
			text, source, items, err := fetchMarketNewsResilient(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, deps.ProjectRoot, market, limit)
			if err != nil {
				return newsUnavailableResult("fetch_market_news", market, "", err)
			}
			if stockNewsNeedsFallback(text) {
				if supplement, _ := webSearchMarketFallback(ctx.GoContext(), deps.Search, market); supplement != "" {
					text = mergeStockNewsText(text, supplement)
					source = source + "+web_search"
				}
			}
			return buildMarketNewsResult(market, text, source, items)
		},
	})
	r.Register(Tool{
		Name: "fetch_stock_news", Description: "拉取个股新闻（经 GeeGooBot→GeeGooData；失败时本地多源 / web_search）。",
		Parameters: map[string]any{
			"type": "object", "required": []string{"code"},
			"properties": map[string]any{
				"code":  map[string]any{"type": "string"},
				"limit": map[string]any{"type": "integer"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			limit := intArg(args, "limit", 8)
			if code == "" {
				return Result{Status: StatusError, Summary: "fetch_stock_news: code required", ExitCode: 1}
			}
			if ctx.DryRun {
				return okDryRun("fetch_stock_news", map[string]any{"code": code, "text": "", "source": "dry-run"})
			}
			text, source, items, err := fetchStockNewsResilient(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, deps.ProjectRoot, code, limit)
			if err != nil {
				return newsUnavailableResult("fetch_stock_news", "", code, err)
			}
			if stockNewsNeedsFallback(text) {
				if supplement, _ := webSearchNewsFallback(ctx.GoContext(), deps.Search, code); supplement != "" {
					text = mergeStockNewsText(text, supplement)
					if source == "" {
						source = "web_search"
					} else {
						source = source + "+web_search"
					}
				}
			}
			if stockNewsNeedsFallback(text) {
				return newsUnavailableResult("fetch_stock_news", "", code, fmt.Errorf("no headlines after GeeGooData and web_search"))
			}
			summary := fmt.Sprintf("fetch_stock_news %s: %d chars", code, len(text))
			if text == "" {
				summary = fmt.Sprintf("fetch_stock_news %s: no items", code)
			}
			return Result{Status: StatusOK, Summary: summary, Data: map[string]any{
				"code": code, "text": text, "items": items, "source": source,
			}}
		},
	})
}

func registerAnalysisTools(r *Registry, deps Deps) {
	registerPromptTemplateTools(r, deps)
	r.Register(Tool{
		Name:        "get_mcp_analysis",
		Description: "执行 MCP 技术面/指数 LLM 分析（经 GeeGooBot :3120，mcp_token 解析用户）。个股：先 get_single_prompt_template(type=tech) 取 prompt_id。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "股票名称（中文或英文名），如 SpaceX",
				},
				"code": map[string]any{
					"type":        "string",
					"description": "股票代码，如 SPCX.US、00700.HK",
				},
				"period": map[string]any{
					"type":        "string",
					"enum":        []any{"daily", "weekly", "hourly"},
					"description": "分析周期",
				},
				"prompt_id": map[string]any{
					"type":        "string",
					"description": "分析模板 ID，来自 get_single_prompt_template；指数可用默认",
				},
				"language": map[string]any{
					"type":        "string",
					"description": "输出语言 cn/en/hk，默认 cn",
				},
			},
			"required": []any{"name", "code", "period"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			name := strArg(args, "name", "")
			code := strArg(args, "code", "")
			period := strArg(args, "period", "daily")
			promptID := strArg(args, "prompt_id", indexPromptID)
			language := strArg(args, "language", "cn")
			if name == "" || code == "" {
				return Result{
					Status: StatusError, ExitCode: 1,
					Summary: "get_mcp_analysis 需要 name 与 code（先 search_code 确认标的）；个股技术面请先 get_single_prompt_template(type=tech) 获取 prompt_id",
				}
			}
			if ctx.DryRun {
				return okDryRun("get_mcp_analysis", map[string]any{
					"code": code, "period": period, "prompt_id": promptID,
					"analysis_result": fmt.Sprintf("[dry-run] analysis for %s", name),
				})
			}
			result, err := getMCPAnalysisResilient(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, name, code, promptID, period, language)
			if err != nil {
				return errResult(err)
			}
			data := map[string]any{
				"code": code, "period": period, "prompt_id": promptID,
				"analysis_result": result.AnalysisResult,
				"model": result.Model, "create_date": result.CreateDate,
			}
			if status, note, _ := ClassifyHTTPPayload("get_mcp_analysis", data, nil); status != StatusOK {
				return Result{Status: status, Summary: note, Data: data}
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("MCP analysis %s (%s)", code, period), Data: data}
		},
	})
	r.Register(Tool{
		Name: "get_capital_flow", Description: "查询主力资金流向（经 GeeGooBot 路由 GeeGooData，DAY 空时自动试 WEEK 并重试）。",
		Parameters: map[string]any{
			"type": "object", "required": []string{"code"},
			"properties": map[string]any{
				"code":   map[string]any{"type": "string", "description": "标的代码"},
				"period": map[string]any{"type": "string", "description": "DAY/WEEK/MONTH，默认 DAY"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			period := strArg(args, "period", "DAY")
			if ctx.DryRun {
				return okDryRun("get_capital_flow", map[string]any{"code": code, "items": []any{}})
			}
			flows, usedPeriod, err := fetchCapitalFlowResilient(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, code, period)
			if err != nil {
				return errResult(err)
			}
			if len(flows) == 0 {
				note := emptyDataNote("get_capital_flow", code)
				return Result{Status: StatusSkip, Summary: note, Data: map[string]any{
					"code": code, "period": usedPeriod, "latest": map[string]any{}, "items": []any{}, "source": "GeeGooBot-mcp-api",
				}}
			}
			items := make([]any, 0, len(flows))
			for _, f := range flows {
				items = append(items, map[string]any{
					"main_in_flow": f.MainInFlow, "in_flow": f.InFlow,
					"time": f.CapitalFlowItemTime,
				})
			}
			latest := map[string]any{"main_in_flow": flows[len(flows)-1].MainInFlow}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("capital flow %s (%s, %d pts)", code, usedPeriod, len(flows)),
				Data: map[string]any{"code": code, "period": usedPeriod, "latest": latest, "items": items, "source": "GeeGooBot-mcp-api"}}
		},
	})
	r.Register(Tool{
		Name: "get_capital_distribution", Description: "查询资金分布（超大/大/中/小单；经 GeeGooBot 路由 GeeGooData，空结果自动重试）。",
		Parameters: map[string]any{
			"type": "object", "required": []string{"code"},
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "标的代码"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return okDryRun("get_capital_distribution", map[string]any{"code": code})
			}
			dist, err := fetchCapitalDistributionResilient(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, code)
			if err != nil {
				return errResult(err)
			}
			formatted := fmt.Sprintf("super_in=%v big_in=%v mid_in=%v small_in=%v update_time=%v",
				dist.CapitalInSuper, dist.CapitalInBig, dist.CapitalInMid, dist.CapitalInSmall, dist.UpdateTime)
			if !capitalDistributionHasData(dist) {
				note := emptyDataNote("get_capital_distribution", code)
				return Result{Status: StatusSkip, Summary: note, Data: map[string]any{
					"code": code, "formatted": formatted, "source": "GeeGooBot-mcp-api",
				}}
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("capital distribution %s", code), Data: map[string]any{
				"code": code, "formatted": formatted, "source": "GeeGooBot-mcp-api",
			}}
		},
	})
	r.Register(Tool{
		Name: "get_bot_yesterday_attitude", Description: "查询 Bot 昨日态度监控结论。必填 bot_id；先 list_*_bots 让用户选定。",
		Parameters: map[string]any{
			"type": "object", "required": []string{"bot_id"},
			"properties": map[string]any{
				"bot_id": map[string]any{"type": "string", "description": "Bot _id"},
				"code":   map[string]any{"type": "string", "description": "可选，展示用"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			botID := strArg(args, "bot_id", "")
			code := strArg(args, "code", "")
			if ctx.DryRun {
				return okDryRun("get_bot_yesterday_attitude", map[string]any{"bot_id": botID, "code": code, "attitude": "neutral"})
			}
			att, err := deps.HTTP.MCP.GetBotYesterdayAttitude(ctx.GoContext(), ctx.MCPToken, botID, "cn")
			if err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("attitude=%s", att.Attitude), Data: map[string]any{
				"bot_id": botID, "code": code, "attitude": att.Attitude, "found": att.Found,
			}}
		},
	})
	r.Register(Tool{
		Name: "get_stock_daily_reports", Description: "按日聚合盘前/盘中/盘后报告。建议传 report_date(YYYY-MM-DD) 与 code。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code":        map[string]any{"type": "string", "description": "标的代码"},
				"report_date": map[string]any{"type": "string", "description": "YYYY-MM-DD，默认今天"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			reportDate := strArg(args, "report_date", today())
			if ctx.DryRun {
				return okDryRun("get_stock_daily_reports", map[string]any{"pre_market": []any{}, "intraday": []any{}, "post_market": []any{}})
			}
			reports, err := deps.HTTP.MCP.GetStockDailyReports(ctx.GoContext(), ctx.MCPToken, code, reportDate)
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
			reports, err := deps.HTTP.MCP.GetStockDailyReports(ctx.GoContext(), ctx.MCPToken, code, reportDate)
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
		Name: "list_today_post_market_reports", Description: "Idempotency check for today's post_market reports.",
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			reportDate := strArg(args, "report_date", today())
			if ctx.DryRun {
				return Result{Status: StatusDryRun, Summary: fmt.Sprintf("dry-run: no existing post_market for %s", code),
					Data: map[string]any{"code": code, "report_date": reportDate, "count": 0, "already_reported": false}}
			}
			reports, err := deps.HTTP.MCP.GetStockDailyReports(ctx.GoContext(), ctx.MCPToken, code, reportDate)
			if err != nil {
				return errResult(err)
			}
			count := len(reports.PostMarket)
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Found %d post_market report(s)", count), Data: map[string]any{
				"code": code, "report_date": reportDate, "count": count,
				"reports": reports.PostMarket, "already_reported": count > 0,
			}}
		},
	})
	r.Register(Tool{
		Name: "recall_yesterday_summary", Description: "读取本地/workspace 昨日盘前摘要；无文件时 skip，可改 get_stock_daily_reports。",
		Parameters: map[string]any{
			"type": "object", "required": []string{"code"},
			"properties": map[string]any{
				"code":        map[string]any{"type": "string"},
				"report_date": map[string]any{"type": "string", "description": "YYYY-MM-DD，默认昨天"},
			},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			code := strArg(args, "code", "")
			if code == "" {
				return Result{Status: StatusError, Summary: "recall_yesterday_summary: code required", ExitCode: 1}
			}
			reportDate := strArg(args, "report_date", yesterday())
			if ctx.DryRun {
				return okDryRun("recall_yesterday_summary", map[string]any{
					"code": code, "report_date": reportDate, "summary": "", "found": false,
				})
			}
			summary, path, found, err := readYesterdayReport(deps.WorkspaceRoot, reportDate, code)
			if err != nil {
				return errResult(err)
			}
			if !found && strings.TrimSpace(ctx.MCPToken) != "" {
				if s, p, ok := recallFromMCPReports(ctx.GoContext(), deps.HTTP.MCP, ctx.MCPToken, code, reportDate, 5); ok {
					summary, path, found = s, p, true
				}
			}
			if !found {
				return Result{
					Status: StatusSkip,
					Summary: fmt.Sprintf("recall_yesterday_summary: no report for %s on %s", code, reportDate),
					Data: map[string]any{
						"code": code, "report_date": reportDate, "summary": "", "found": false, "implemented": true,
					},
				}
			}
			return Result{
				Status: StatusOK,
				Summary: fmt.Sprintf("recall_yesterday_summary %s (%s): %d chars", code, reportDate, len(summary)),
				Data: map[string]any{
					"code": code, "report_date": reportDate, "summary": summary, "path": path,
					"found": true, "implemented": true,
				},
			}
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
			query := strArg(args, "query", "")
			limit := intArg(args, "limit", 5)
			if limit < 1 {
				limit = 1
			}
			if limit > 20 {
				limit = 20
			}
			if deps.Memory != nil {
				res, err := deps.Memory.Recall(ctx.GoContext(), memport.RecallQuery{
					Kind: memport.RecallSession, Query: query,
					ExcludeSessionID: ctx.SessionID, Limit: limit, ScanLimit: 30,
				})
				if err != nil {
					return errResult(err)
				}
				if len(res.Hits) == 0 {
					return Result{
						Status: StatusOK, Summary: "No matching past chat sessions",
						Data: map[string]any{"count": 0, "matches": []any{}},
					}
				}
				top := res.Hits[0]
				summary := fmt.Sprintf("Found %d session(s); latest: %s (%s)", len(res.Hits), top.Snippet, top.ID)
				return Result{Status: StatusOK, Summary: summary, Data: res.Data}
			}
			if ctx.StateStore == nil {
				return Result{Status: StatusError, Summary: "state_store not configured", ExitCode: 1}
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
			result, err := deps.HTTP.MCP.CreatePreMarketReport(ctx.GoContext(), ctx.MCPToken, args)
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

func firstStringArg(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := strArg(args, key, ""); v != "" {
			return v
		}
	}
	return ""
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

func yesterday() string { return time.Now().AddDate(0, 0, -1).Format("2006-01-02") }

func readYesterdayReport(workspaceRoot, reportDate, code string) (summary, path string, found bool, err error) {
	guard, err := infra.NewWorkspaceGuard(workspaceRoot)
	if err != nil {
		return "", "", false, err
	}
	rel := fmt.Sprintf("reports/%s/%s-premarket.md", reportDate, code)
	path, err = guard.Resolve(rel)
	if err != nil {
		return "", "", false, err
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", rel, false, nil
		}
		return "", "", false, readErr
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", rel, false, nil
	}
	return truncateRecall(text, 4000), path, true, nil
}

func firstPreMarketText(items []map[string]any) string {
	for _, m := range items {
		for _, key := range []string{"content", "report_content", "summary", "text"} {
			if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func truncateRecall(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func emptyDataNote(tool, code string) string {
	base := fmt.Sprintf("%s: 无可用数据", tool)
	switch tool {
	case "get_position":
		return base + "（富途账户未配置或该标的无持仓；盘中逐笔请用 get_ticker）"
	case "get_ticker", "get_broker":
		return base + "（富途 OpenD 未配置或非交易时段；现价请用 get_current_price）"
	case "get_capital_flow", "get_capital_distribution":
		if isAShare(code) {
			return base + "（经 Bot→GeeGooData CN 节点；无数据时检查 FutuOpenD 与 Bot 路由配置）"
		}
		return base + "（经 Bot→GeeGooData HK/US 节点；非交易时段或数据源无记录）"
	default:
		if code != "" {
			return fmt.Sprintf("%s %s: API 返回成功但数据为空", tool, code)
		}
		return base
	}
}

func newsUnavailableResult(tool, market, code string, err error) Result {
	if errors.Is(err, newsrunner.ErrUnavailable) {
		data := map[string]any{"text": "", "source": "unavailable"}
		if market != "" {
			data["market"] = market
			data["items"] = []any{}
		}
		if code != "" {
			data["code"] = code
		}
		return Result{
			Status:  StatusSkip,
			Summary: tool + ": 新闻获取失败（GeeGooData / 本地回退均不可用）",
			Data:    data,
		}
	}
	return errResult(err)
}

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

func registerPromptTemplateTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name:        "get_single_prompt_template",
		Description: "获取单项分析 Prompt 模板列表。调用 get_mcp_analysis 前必须先取 prompt_id；个股信号/技术面用 type=tech。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":        "string",
					"enum":        []any{"index", "tech", "fundamental"},
					"description": "模板类型：个股技术面/信号趋势用 tech，指数用 index，基本面用 fundamental",
				},
				"period": map[string]any{
					"type":        "string",
					"description": "可选周期过滤，如 daily、weekly、hourly",
				},
			},
			"required": []any{"type"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			type_ := strings.ToLower(strings.TrimSpace(strArg(args, "type", "")))
			if type_ == "" {
				type_ = "tech"
			}
			switch type_ {
			case "index", "tech", "fundamental":
			default:
				return Result{
					Status: StatusError, ExitCode: 1,
					Summary: "get_single_prompt_template 需要 type=index|tech|fundamental",
				}
			}
			period := strings.TrimSpace(strArg(args, "period", ""))
			if ctx.DryRun {
				data := map[string]any{
					"type": type_, "items": []any{
						map[string]any{"prompt_id": "dry-run-prompt", "name": "dry-run", "period": period},
					},
				}
				if period != "" {
					data["period"] = period
				}
				return okDryRun("get_single_prompt_template", data)
			}
			if strings.TrimSpace(ctx.MCPToken) == "" {
				return Result{Status: StatusError, Summary: "缺少 mcp_token：请运行 geegoo setup 配置", ExitCode: 1}
			}
			body := map[string]any{"mcp_token": ctx.MCPToken, "type": type_}
			if period != "" {
				body["period"] = period
			}
			data, err := deps.HTTP.MCP.PostDirect(ctx.GoContext(), "/getSinglePromptTemplate", body)
			if err != nil {
				return errResult(err)
			}
			normalized, summary := normalizeHTTPResponse("get_single_prompt_template", data)
			if status, note, _ := ClassifyHTTPPayload("get_single_prompt_template", normalized, nil); status != StatusOK {
				return Result{Status: status, Summary: note, Data: normalized}
			}
			return Result{Status: StatusOK, Summary: summary, Data: normalized}
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
