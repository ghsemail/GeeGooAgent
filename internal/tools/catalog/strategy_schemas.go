package catalog

func loopbackStrategyParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{"type", "code", "frequency", "fund", "months_back"},
		"properties": map[string]any{
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{"grid", "dca"},
				"description": "策略类型：grid 网格 / dca 定投",
			},
			"strategy_type": map[string]any{
				"type":        "string",
				"enum":        []string{"grid", "dca"},
				"description": "同 type，二选一即可",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "标的代码，如 00700.HK",
			},
			"frequency": map[string]any{
				"type":        "string",
				"description": "K 线周期：grid 常用 5m，dca 常用 60m",
			},
			"fund": map[string]any{
				"type":        "number",
				"description": "初始资金",
			},
			"months_back": map[string]any{
				"type":        "integer",
				"description": "回测月数，与 generate_* 保持一致",
			},
			"base_order_size": map[string]any{
				"type":        "number",
				"description": "每单股数，默认 100",
			},
			"grid_param": map[string]any{
				"type":        "object",
				"description": "type=grid 时必填；可直接用 generate_grid_strategy 返回的 param",
				"properties": map[string]any{
					"upper_limit_price": map[string]any{"type": "number"},
					"lower_limit_price": map[string]any{"type": "number"},
					"grid_num":          map[string]any{"type": "integer"},
				},
				"required": []string{"upper_limit_price", "lower_limit_price", "grid_num"},
			},
			"signal": map[string]any{
				"type":        "array",
				"description": "type=dca 时必填；用 generate_dca_strategy 返回的 signal.buy_signal 数组",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"index": map[string]any{"type": "string"},
						"type":  map[string]any{"type": "string", "description": "signal 或 flag"},
						"param": map[string]any{"type": "object"},
					},
				},
			},
			"sl_tp": map[string]any{
				"type":        "object",
				"description": "type=dca 时必填；由 generate_dca_strategy 的 dynamicParam 或 fixedParam 组装，顶层需 type=fix 或 dynamic",
				"properties": map[string]any{
					"type": map[string]any{"type": "string", "enum": []string{"fix", "dynamic"}},
					"tp":   map[string]any{"type": "object"},
					"sl":   map[string]any{"type": "object"},
				},
				"required": []string{"type", "tp", "sl"},
			},
		},
	}
}

func generateGridStrategyParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name"},
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "股票代码",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "股票名称",
			},
			"months_back": map[string]any{
				"type":        "integer",
				"description": "分析/回测月数，默认 1",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "cn / en / hk / all",
			},
		},
	}
}

func generateDCAStrategyParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "name", "signal_id"},
		"properties": map[string]any{
			"code": map[string]any{"type": "string"},
			"name": map[string]any{"type": "string"},
			"signal_id": map[string]any{
				"type":        "string",
				"description": "来自 get_index_signals 或 get_signal_combinations",
			},
			"months_back": map[string]any{"type": "integer"},
			"language":    map[string]any{"type": "string"},
		},
	}
}
