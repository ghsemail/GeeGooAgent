package eval

// defaultExpectReplyForTurnID returns semantic acceptance criteria per turn_plan case.
func defaultExpectReplyForTurnID(turnID string) ExpectReplySpec {
	switch turnID {
	case "stock_price":
		return ExpectReplySpec{
			Rubric:    "应给出腾讯控股的现价或行情信息（价格、涨跌幅等），语气自然。",
			MustCover: []string{"腾讯"},
		}
	case "stock_price_trend":
		return ExpectReplySpec{
			Rubric:    "应分析腾讯最近一个月的价格走势（趋势、涨跌、关键价位等），走 MCP 深度分析而非只报现价。",
			MustCover: []string{"腾讯", "走势"},
		}
	case "stock_technical_chain":
		return ExpectReplySpec{
			Rubric:    "在已查腾讯股价的基础上，应补充 K 线图或技术面分析，而非只重复报价。",
			MustCover: []string{"腾讯"},
		}
	case "stock_symbol_switch":
		return ExpectReplySpec{
			Rubric:    "应识别用户切换到贵州茅台，并给出茅台相关分析或行情，而非继续只聊中际旭创。",
			MustCover: []string{"茅台"},
		}
	case "stock_colloquial_ref":
		return ExpectReplySpec{
			Rubric:    "应理解「它」指代上一轮的中际旭创，并继续给出该标的信号趋势或技术面解读，而非换标的或跑回测。",
			MustCover: []string{"中际", "信号"},
		}
	case "subagent_multi_stock_price":
		return ExpectReplySpec{
			Rubric:    "应分别给出腾讯与阿里巴巴最近股价或行情信息，并在同一答复中做简要对比或并列总结；不应只分析其中一个标的。",
			MustCover: []string{"腾讯", "阿里"},
		}
	case "signal_catalog_list":
		return ExpectReplySpec{
			Rubric:    "应列出或摘要用户可用的信号/组合策略，语气自然，不应直接跑回测或 probe。",
			MustCover: []string{"信号"},
			MustNot:   []string{"开始回测"},
		}
	case "signal_list_then_probe":
		return ExpectReplySpec{
			Rubric:    "在已分析腾讯并讨论适合策略后，应识别回测意图并针对腾讯发起或汇报回测进度/结果，而非只做静态分析或 probe 买卖点。",
			MustCover: []string{"腾讯", "回测"},
		}
	case "signal_probe_direct":
		return ExpectReplySpec{
			Rubric:    "应对中际旭创执行或汇报买卖点探测结果，包含信号方向或暂无信号说明。",
			MustCover: []string{"中际"},
		}
	case "backtest_explicit":
		return ExpectReplySpec{
			Rubric:    "应确认 SAR+MACD 回测小米并已发起或汇报回测结果/进度，而非仅做股价查询。",
			MustCover: []string{"小米", "回测"},
		}
	case "backtest_colloquial":
		return ExpectReplySpec{
			Rubric:    "应识别回测意图并针对中际旭创发起或说明回测，而非只做静态分析。",
			MustCover: []string{"中际", "回测"},
		}
	case "analysis_then_backtest":
		return ExpectReplySpec{
			Rubric:    "在已分析小米的语境下，应发起 SAR+MACD 回测或明确回测执行与结果摘要。",
			MustCover: []string{"回测"},
		}
	case "strategy_list_then_backtest":
		return ExpectReplySpec{
			Rubric:    "在用户已询问可回测策略后，应确认 SAR+MACD 并对中际旭创执行回测或汇报回测结果。",
			MustCover: []string{"中际", "回测"},
		}
	case "ambiguous_bare_macd":
		return ExpectReplySpec{
			Rubric:  "用户已选定具体 MACD 组合信号（如 SAR+MACD）后，应基于该信号/策略说明讲解日常用法、适用场景或注意事项，而非直接测点或回测。",
			MustCover: []string{"MACD", "SAR"},
			MustNot:   []string{"回测已完成", "开始回测"},
		}
	case "compound_analysis_backtest":
		return ExpectReplySpec{
			Rubric:    "复合意图下用户选择「先只做分析」后，应针对中际旭创给出分析或行情解读，不应未经确认直接跑回测。",
			MustCover: []string{"中际"},
			MustNot:   []string{"开始回测"},
		}
	case "stock_quote_ambiguous":
		return ExpectReplySpec{
			Rubric:    "用户已选择「只要当前价」后，应给出腾讯控股的现价或简要行情（价格、涨跌幅等），而非展开 MCP 深度分析或回测。",
			MustCover: []string{"腾讯"},
			MustNot:   []string{"开始回测"},
		}
	case "chat_definition":
		return ExpectReplySpec{
			Rubric:    "应用通俗语言解释 MACD 指标含义，不应调用行情或回测工具。",
			MustCover: []string{"MACD"},
		}
	case "chat_signal_quality":
		return ExpectReplySpec{
			Rubric:    "在已做买卖点探测后，应对信号是否靠谱做定性说明或限制条件，走闲聊/解释而非再次 probe。",
		}
	case "bot_reminder_list":
		return ExpectReplySpec{
			Rubric:    "应列出或说明用户当前的 Reminder/定时提醒，信息结构清晰。",
			MustCover: []string{"reminder"},
		}
	case "bot_grid_pnl":
		return ExpectReplySpec{
			Rubric:    "应汇报腾讯网格 Bot 的盈亏或状态，与 grid bot 相关。",
			MustCover: []string{"网格"},
		}
	case "bot_smarttrade_list":
		return ExpectReplySpec{
			Rubric:    "应列出 SmartTrade 实例或说明暂无，回应「有哪些 SmartTrade」。",
			MustCover: []string{"SmartTrade"},
		}
	case "backtest_history":
		return ExpectReplySpec{
			Rubric:    "在刚跑完回测的语境下，应查询或摘要上次/本次回测结果，而非重新发起无关分析。",
			MustCover: []string{"回测"},
		}
	case "report_lookup":
		return ExpectReplySpec{
			Rubric:    "应返回或摘要今日盘前报告内容，或说明暂无盘前报告。",
			MustCover: []string{"盘前"},
		}
	case "knowledge_lookup":
		return ExpectReplySpec{
			Rubric:    "应基于知识库讲解 4H MACD，内容偏教学/知识而非直接下单建议。",
			MustCover: []string{"MACD"},
		}
	case "news_lookup":
		return ExpectReplySpec{
			Rubric:    "应提供市场或财经新闻摘要，至少列举若干条或说明来源。",
			MustCover: []string{"新闻"},
		}
	case "dca_grid_backtest":
		return ExpectReplySpec{
			Rubric:    "应识别 DCA/定投回测意图并给出方案、参数确认或回测执行说明。",
			MustCover: []string{"DCA"},
		}
	default:
		return ExpectReplySpec{}
	}
}
