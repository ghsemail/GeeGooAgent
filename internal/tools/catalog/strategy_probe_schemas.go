package catalog

func signalRuleItemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"index": stringProp("指标名，如 MACD、RSI、SAR；定制策略用 custom.index（如 Macd4HRhythm）"),
			"type":  stringProp("signal 或 flag"),
			"param": objectProp("指标参数，定制策略按 get_custom_strategy_definitions 的 param_schema"),
		},
		"required": []string{"index", "type"},
	}
}

func probeBotSignalSeriesParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "frequency", "buy_signal"},
		"properties": map[string]any{
			"code":      stringProp("标的代码，如 00700.HK、AAPL.US"),
			"frequency": stringProp("K 线周期：5m / 15m / 60m / daily"),
			"buy_signal": map[string]any{
				"type":        "array",
				"description": "买入规则链；每项含 index、type、param",
				"minItems":    float64(1),
				"items":       signalRuleItemSchema(),
			},
			"sell_signal": map[string]any{
				"type":        "array",
				"description": "卖出规则链；可空",
				"items":       signalRuleItemSchema(),
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "K 线根数，默认按 months_back 推算，最大 800",
			},
			"months_back": intProp("回溯月数，默认 3；与 trading_operation 策略开发/回测一致"),
		},
	}
}

func probeBotSignalParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "frequency", "buy_signal"},
		"properties": map[string]any{
			"code":        stringProp("标的代码"),
			"frequency":   stringProp("K 线周期：5m / 15m / 60m / daily"),
			"buy_signal":  map[string]any{"type": "array", "minItems": float64(1), "items": signalRuleItemSchema()},
			"sell_signal": map[string]any{"type": "array", "items": signalRuleItemSchema()},
			"at":          stringProp("可选，指定 bar 时间；默认最后一根"),
		},
	}
}

func getIndicatorSeriesParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "frequency", "index"},
		"properties": map[string]any{
			"code":        stringProp("标的代码"),
			"frequency":   stringProp("K 线周期"),
			"index":       stringProp("指标名，如 SAR、BBAND"),
			"role":        stringProp("可选：sl / tp，动态止盈止损用"),
			"limit":       intProp("序列长度"),
			"months_back": intProp("回溯月数"),
			"param":       objectProp("指标参数"),
		},
	}
}

func listStrategyBacktestLogsParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit":          intProp("条数，默认 100"),
			"skip":           intProp("跳过条数"),
			"code":           stringProp("按标的筛选，如 1810.HK"),
			"strategy_label": stringProp("按策略名筛选"),
			"date_from":      stringProp("起始日期 YYYY-MM-DD"),
			"date_to":        stringProp("结束日期 YYYY-MM-DD"),
		},
	}
}

func getStrategyBacktestLogParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"log_id"},
		"properties": map[string]any{
			"log_id": stringProp("list_strategy_backtest_logs 返回的 log_id"),
		},
	}
}
