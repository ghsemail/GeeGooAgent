package chatui

// displayToolName maps internal tool ids to Hermes-style labels in progress output.
func displayToolName(name string) string {
	switch name {
	case "recall":
		return "session_search"
	case "web_search":
		return "web_search"
	case "get_ticker":
		return "get_ticker" // MCP /getTicker — 盘中逐笔，非 get_current_price
	case "get_current_price":
		return "get_current_price"
	default:
		return name
	}
}

func toolEmoji(name string) string {
	switch name {
	case "search_code", "recall", "web_search":
		return "🔍"
	case "get_current_price", "get_ticker":
		return "💹"
	case "get_position":
		return "📊"
	case "check_trading_day":
		return "📅"
	case "fetch_market_news", "fetch_stock_news":
		return "📰"
	case "get_mcp_analysis":
		return "📊"
	case "get_capital_flow", "get_capital_distribution":
		return "💰"
	case "get_stock_daily_reports", "list_today_reports":
		return "📝"
	case "write_execution_log":
		return "📋"
	default:
		return "⚡"
	}
}
