package diagram

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type namedStep struct {
	Name   string
	Title  string
	Tool   string
	Params string
	Kind   string // prelude | perstock | chat
}

type stepDisplay struct {
	Title string
	Tool  string
	Hint  string
}

var knownStepDisplay = map[string]stepDisplay{
	"cognition_pick":          {"解析策略名称", "", "strategy"},
	"cognition_read_catalog":  {"读策略库", "catalog", "组合 / 指标 / 定制 / definitions"},
	"cognition_compose":       {"LLM 合成策略档案", "llm", "场景 / 参数 / 信号逻辑"},
	"cognition_save_kb":       {"写入知识库", "save_strategy_knowledge", "策略档案/"},
	"cognition_verify_kb":     {"读回验证", "search_knowledge", "策略档案/"},
	"summarize":               {"汇总输出", "", ""},
	"dev_pick":                {"解析策略名称", "", "strategy"},
	"dev_ensure_archive":      {"确保策略档案", "search_knowledge", "无则自动生成"},
	"resolve_symbol":          {"解析标的", "", "code"},
	"pick_strategies":         {"选取策略", "", "catalog"},
	"probe_foreach":           {"逐个探测信号", "probe_bot_signal_series", ""},
	"resolve_context":         {"解析标的与策略", "", "code / strategy"},
	"baseline_probe":          {"默认参数探测", "probe_bot_signal_series", ""},
	"pick_param_variants":     {"生成参数组合", "", "months_back / frequency"},
	"check_trading_day":       {"检查交易日", "check_trading_day", ""},
	"get_report_bot_codes":    {"读取订阅标的", "get_report_bot_codes", ""},
	"get_market_premarket_report": {"读取市场盘前报告", "get_market_premarket_report", ""},
	"list_today_reports":      {"列出今日盘前报告", "list_today_reports", ""},
	"list_today_postmarket_stock": {"列出今日盘后报告", "list_today_stock_postmarket_reports", ""},
	"stock_news":              {"个股新闻", "fetch_stock_news", ""},
	"capital_flow":            {"资金流向", "get_capital_flow", ""},
	"capital_distribution":    {"资金分布", "get_capital_distribution", ""},
	"weekly_analysis":         {"周线分析", "get_mcp_analysis", ""},
	"bot_attitude":            {"昨日态度", "get_bot_yesterday_attitude", ""},
	"save_local_report":       {"保存本地报告", "save_local_report", ""},
	"create_stock_premarket_report": {"写入个股盘前报告", "create_stock_premarket_report", ""},
	"create_market_premarket_report": {"写入市场盘前报告", "create_market_premarket_report", ""},
	"create_stock_intraday_report": {"写入盘中报告", "create_stock_intraday_report", ""},
	"create_stock_postmarket_report": {"写入盘后报告", "create_stock_postmarket_report", ""},
	"phase_a_complete":        {"Phase A 完成", "write_execution_log", ""},
	"stock_complete":          {"个股步骤完成", "write_execution_log", ""},
	"hourly_analysis_bundle":  {"小时级分析包", "get_hourly_analysis_bundle", ""},
	"hourly_price_analysis":   {"小时级价格分析", "get_mcp_analysis", ""},
	"current_price":           {"现价", "get_current_price", ""},
	"get_position":            {"持仓", "get_position", ""},
	"read_stock_premarket":    {"读取个股盘前", "get_stock_daily_reports", ""},
	"bot_log":                 {"机器人日志", "get_bot_log_by_type", ""},
}

var (
	reSkillTablePhase = regexp.MustCompile("(?m)^\\|\\s*`([^`]+)`\\s*\\|\\s*([^|]+)\\|")
	reSkillDashPhase  = regexp.MustCompile("(?m)^(?:\\d+\\.\\s*)?`([^`]+)`\\s*[—–-]\\s*(.+)$")
)

func parsePhaseTitles(skillMD string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(skillMD) == "" {
		return out
	}
	for _, m := range reSkillTablePhase.FindAllStringSubmatch(skillMD, -1) {
		key := strings.TrimSpace(m[1])
		title := compactTitle(m[2])
		if key != "" && title != "" {
			out[key] = title
		}
	}
	for _, m := range reSkillDashPhase.FindAllStringSubmatch(skillMD, -1) {
		key := strings.TrimSpace(m[1])
		if _, ok := out[key]; ok {
			continue
		}
		title := compactTitle(m[2])
		if key != "" && title != "" {
			out[key] = title
		}
	}
	return out
}

func compactTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.IndexAny(s, "。；;"); i > 0 {
		s = s[:i]
	}
	return truncateLabel(s, 18)
}

func applyStepDisplay(step namedStep, titles map[string]string) namedStep {
	if known, ok := knownStepDisplay[step.Name]; ok {
		step.Title = known.Title
		if step.Tool == "" || step.Tool == "(chat workflow)" {
			step.Tool = known.Tool
		}
		if step.Params == "" {
			step.Params = known.Hint
		}
	}
	if step.Title == "" {
		if title := titles[step.Name]; title != "" {
			step.Title = title
		}
	}
	step.Title = displayArchiveCopy(step.Title)
	if strings.HasPrefix(step.Name, "index_") {
		if step.Title == "" {
			step.Title = "指数分析"
		}
		if step.Params == "" {
			step.Params = strings.TrimPrefix(step.Name, "index_")
		}
	}
	if strings.HasPrefix(step.Name, "market_news_") {
		if step.Title == "" {
			step.Title = "市场新闻"
		}
		if step.Params == "" {
			step.Params = strings.ToUpper(strings.TrimPrefix(step.Name, "market_news_"))
		}
	}
	if step.Title == "" {
		step.Title = step.Name
	}
	return step
}

func formatArgs(raw any) string {
	args, ok := raw.(map[string]any)
	if !ok || len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := args[k]
		if v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) == "" {
				continue
			}
			parts = append(parts, k+"="+truncateLabel(t, 16))
		case bool:
			parts = append(parts, fmt.Sprintf("%s=%t", k, t))
		case int, int32, int64, float32, float64:
			parts = append(parts, fmt.Sprintf("%s=%v", k, t))
		default:
			continue
		}
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, " · ")
}

func nodeLabelOf(step namedStep) string {
	if strings.TrimSpace(step.Title) != "" {
		return step.Title
	}
	return step.Name
}

func nodeSublabelOf(step namedStep) string {
	parts := make([]string, 0, 3)
	if step.Name != "" && step.Name != step.Title {
		parts = append(parts, step.Name)
	}
	if tool := strings.TrimSpace(step.Tool); tool != "" && tool != "(chat workflow)" && tool != step.Name && tool != step.Title {
		parts = append(parts, tool)
	}
	if params := strings.TrimSpace(step.Params); params != "" && params != step.Name && params != step.Tool {
		parts = append(parts, params)
	}
	return truncateLabel(displayArchiveCopy(strings.Join(parts, " · ")), 36)
}

func displayArchiveCopy(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "策略认知", "策略档案")
	s = strings.ReplaceAll(s, "（兼容 策略档案/）", "")
	s = strings.ReplaceAll(s, "(兼容 策略档案/)", "")
	return strings.Join(strings.Fields(s), " ")
}

func displayChatTriggers(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := displayArchiveCopy(raw)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
