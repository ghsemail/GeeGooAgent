package tools

// Core chat tools kept in schema when defer_load_tools is enabled (~15).
var coreChatToolNames = []string{
	"search_code",
	"recall",
	"clarify",
	"get_current_price",
	"delegate_task",
	"delegate_tasks",
	"web_search",
	"check_trading_day",
	"get_ticker",
	"get_position",
	"fetch_stock_news",
	"fetch_market_news",
	"manage_memory",
	"discover_tools",
	"activate_toolset",
}

// CoreChatToolSet returns the core tool name set.
func CoreChatToolSet() map[string]struct{} {
	out := make(map[string]struct{}, len(coreChatToolNames))
	for _, name := range coreChatToolNames {
		out[name] = struct{}{}
	}
	return out
}

// CoreChatToolNames returns sorted core tool names.
func CoreChatToolNames() []string {
	out := append([]string(nil), coreChatToolNames...)
	return sortedStringSlice(out)
}

func sortedStringSlice(items []string) []string {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}
