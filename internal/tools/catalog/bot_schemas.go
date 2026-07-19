package catalog

type botKind string

const (
	botKindDCA          botKind = "dca"
	botKindGrid         botKind = "grid"
	botKindSmartTrade   botKind = "smart_trade"
	botKindHDG          botKind = "hdg"
	botKindDCAReminder  botKind = "dca_reminder"
	botKindGridReminder botKind = "grid_reminder"
	botKindSmartReminder botKind = "smart_reminder"
)

func botCRUD(slug string, kind botKind, labelZh, create, update, delete, list, log string) []HTTPSpec {
	return []HTTPSpec{
		{
			Name:         "create_" + slug,
			Description:  botCreateDescription(kind, labelZh),
			Path:         create,
			MergePayload: true,
			Parameters:   botCreateParameters(kind),
		},
		{
			Name:         "update_" + slug,
			Description:  "更新" + labelZh + "。必填 bot_id（从 list 取得）；只传需修改字段。写操作需用户确认。",
			Path:         update,
			MergePayload: true,
			Parameters:   botUpdateParameters(),
		},
		{
			Name:        "delete_" + slug,
			Description: "删除" + labelZh + "。必填 bot_id。写操作需用户确认。",
			Path:        delete,
			Parameters:  botDeleteParameters(),
		},
		{
			Name:        "list_" + slug + "s",
			Description: "列出当前用户全部" + labelZh + "（含 bot_id、botname、code、stock_name、switch）。",
			Path:        list,
			Parameters:  emptyObjectSchema(),
		},
		{
			Name:        "get_" + slug + "_log",
			Description: "查询" + labelZh + "运行日志。必填 bot_id。",
			Path:        log,
			Parameters:  botLogParameters(),
		},
	}
}

func botCreateDescription(kind botKind, labelZh string) string {
	base := "创建" + labelZh + "（写入 Mongo；GeeGooBot 侧暂无自动 scheduler，创建后需 TradingBot 或后续调度才实盘运行）。写操作需用户确认。"
	switch kind {
	case botKindGrid:
		return base + " 推荐链路：search_code → generate_grid_strategy → loopback 可选 → 用户确认 botname/lot_size → create_grid_bot（grid=generate 的 param，frequency 默认 5m）。"
	case botKindDCA:
		return base + " 推荐链路：选 signal_id → generate_dca_strategy → loopback 可选 → 用户确认 botname/lot_size → create_dca_bot（signal/tp/sl 由 generate 结果映射）。"
	case botKindGridReminder:
		return base + " 仅提醒不下单。grid 参数可来自 generate_grid_strategy 的 param；frequency 默认 5m。"
	case botKindDCAReminder:
		return base + " 仅提醒不下单。signal.buy_signal 可来自 generate_dca_strategy 的 signal。"
	case botKindSmartTrade, botKindHDG:
		return base + " 创建前先 search_code 确认标的，向用户确认 botname、frequency 与策略参数。"
	case botKindSmartReminder:
		return base + " Smart 提醒 Bot；创建前确认 botname、code、frequency。"
	default:
		return base
	}
}

func botCreateParameters(kind botKind) map[string]any {
	props := map[string]any{
		"botname":     stringProp("机器人名称，全局唯一"),
		"stock_name":  stringProp("标的名称"),
		"code":        stringProp("标的代码，如 00700.HK"),
		"frequency":   stringProp("检查频率，如 5m、60m"),
		"lot_size":    intProp("每手股数，常用 100"),
	}
	required := []string{"botname", "stock_name", "code"}

	switch kind {
	case botKindGrid, botKindGridReminder:
		props["grid"] = gridParamSchema()
		required = append(required, "grid")
		if kind == botKindGrid {
			props["order_size"] = objectProp("可选；lot_size 会自动生成 base_order_size")
		}
	case botKindDCA:
		props["signal"] = dcaSignalSchema("买卖信号；buy_signal 数组来自 generate_dca_strategy.signal.buy_signal")
		props["tp"] = objectProp("止盈：tp_mode=fix|dynamic，fix_tp 或 tp_dynamic_index 等")
		props["sl"] = objectProp("止损：sl_mode=fix|dynamic，fix_sl 或 sl_dynamic_index 等")
		required = append(required, "signal")
	case botKindDCAReminder:
		props["signal"] = dcaSignalSchema("提醒信号；buy_signal 来自 generate_dca_strategy")
		required = append(required, "signal")
	case botKindSmartTrade:
		props["strategy"] = objectProp("SmartTrade 策略配置（按用户意图填写）")
	case botKindHDG:
		props["hedge"] = objectProp("HDG 对冲配置（按用户意图填写）")
	}

	return map[string]any{
		"type":       "object",
		"required":   required,
		"properties": props,
	}
}

func botUpdateParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"bot_id"},
		"properties": map[string]any{
			"bot_id":     stringProp("要更新的 Bot _id"),
			"botname":    stringProp("可选"),
			"stock_name": stringProp("可选"),
			"code":       stringProp("可选"),
			"frequency":  stringProp("可选"),
			"grid":       objectProp("GRID 网格配置"),
			"signal":     objectProp("DCA 信号配置"),
			"tp":         objectProp("止盈配置"),
			"sl":         objectProp("止损配置"),
			"order_size": objectProp("仓位配置"),
			"switch":     map[string]any{"type": "boolean", "description": "启用/暂停"},
		},
	}
}

func botDeleteParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"bot_id"},
		"properties": map[string]any{
			"bot_id": stringProp("要删除的 Bot _id"),
		},
	}
}

func botLogParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"bot_id"},
		"properties": map[string]any{
			"bot_id": stringProp("Bot _id"),
		},
	}
}

func stringProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func objectProp(desc string) map[string]any {
	return map[string]any{"type": "object", "description": desc}
}

func gridParamSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": "网格参数：upper_limit_price、lower_limit_price、grid_num（来自 generate_grid_strategy.param）",
		"required":    []string{"upper_limit_price", "lower_limit_price", "grid_num"},
		"properties": map[string]any{
			"upper_limit_price": map[string]any{"type": "number"},
			"lower_limit_price": map[string]any{"type": "number"},
			"grid_num":          map[string]any{"type": "integer"},
		},
	}
}

func dcaSignalSchema(desc string) map[string]any {
	return map[string]any{
		"type":        "object",
		"description": desc,
		"required":    []string{"buy_signal"},
		"properties": map[string]any{
			"buy_signal": map[string]any{
				"type":        "array",
				"description": "来自 generate_dca_strategy.signal.buy_signal",
				"minItems":    float64(1),
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"index": map[string]any{"type": "string"},
						"type":  map[string]any{"type": "string"},
						"param": map[string]any{"type": "object"},
					},
				},
			},
		},
	}
}

func emptyObjectSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
