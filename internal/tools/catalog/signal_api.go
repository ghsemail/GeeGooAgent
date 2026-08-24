package catalog

// UsesSignalAPI reports tools that call GeeGooSignal signal-api :3200 directly.
func UsesSignalAPI(name string) bool {
	switch name {
	case "search_code",
		"loopback_strategy",
		"probe_bot_signal",
		"probe_bot_signal_series",
		"run_strategy_backtest",
		"get_indicator_series",
		"list_strategy_backtest_logs",
		"get_strategy_backtest_log":
		return true
	default:
		return false
	}
}
