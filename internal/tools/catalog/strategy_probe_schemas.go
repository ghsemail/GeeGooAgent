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
				"description": "可选：显式 K 线根数（最大 800）。默认不传，由 months_back 自动换算；仅 Web 对齐「1天/1周」或特殊下限才用",
			},
			"months_back": intProp("回溯月数（优先使用）。默认 3；1月→1、3月→3；服务端按 frequency 换算 limit"),
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

func runStrategyBacktestParameters() map[string]any {
	props := probeBotSignalSeriesParameters()["properties"].(map[string]any)
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	out["strategy_label"] = stringProp("策略展示名，如 4小时MACD市场节奏")
	out["strategy_kind"] = stringProp("indicator / combination / custom，默认 custom")
	out["period"] = stringProp("回溯周期 UI 标签：1m/2m/3m 或 2w，默认 1m")
	out["fund"] = intProp("初始资金，默认 100000")
	out["base_order_size"] = intProp("每次买入股数，默认 100")
	out["stock_name"] = stringProp("标的名称，可选")
	out["market"] = stringProp("市场标签 HK/CN/US，可选")
	out["trade_config"] = objectProp("交易参数（止盈止损/仓位/风控），缺省用 playbook 默认")
	out["strategy_ids"] = map[string]any{
		"type":        "array",
		"description": "关联 signal_id 列表，可选",
		"items":       map[string]any{"type": "string"},
	}
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "frequency", "buy_signal"},
		"properties": out,
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
