package eval

// defaultExpectReplyForTurnID returns semantic acceptance criteria per turn_plan case.
func defaultExpectReplyForTurnID(turnID string) ExpectReplySpec {
	switch turnID {
	case "stock_price":
		return ExpectReplySpec{
			Rubric:    "应给出腾讯控股的股价或行情信息（现价、涨跌幅等），语气自然。",
			MustCover: []string{"腾讯"},
		}
	case "stock_technical_chain":
		return ExpectReplySpec{
			Rubric:    "在已查腾讯股价的基础上，应补充技术面/K 线或价格走势分析，而非只重复报价。",
			MustCover: []string{"腾讯"},
		}
	case "stock_symbol_switch":
		return ExpectReplySpec{
			Rubric:    "应识别用户切换到贵州茅台，并给出茅台相关分析或行情，而非继续只聊中际旭创。",
			MustCover: []string{"茅台"},
		}
	case "stock_colloquial_ref":
		return ExpectReplySpec{
			Rubric:    "应理解「这边呢」指代中际旭创，并继续给出该标的的分析或行情。",
			MustCover: []string{"中际"},
		}
	case "signal_list_then_probe":
		return ExpectReplySpec{
			Rubric:    "在用户已看过策略列表后，应确认 SAR+MACD 组合并对中际旭创做买卖点探测（含买/卖/暂无信号等），不应直接跑完整回测。",
			MustCover: []string{"中际", "买"},
			MustNot:   []string{"开始回测"},
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
			Rubric:    "应对模糊的 MACD 信号问题给出澄清选项（测点/回测/讲解指标等），不应直接执行回测或测点工具。",
			MustNot:   []string{"回测已完成", "开始回测"},
		}
	case "compound_analysis_backtest":
		return ExpectReplySpec{
			Rubric:    "一句话同时含分析与回测时，应澄清用户想先做哪一步或分步确认，不应未经确认直接全套执行。",
		}
	case "chat_definition":
		return ExpectReplySpec{
			Rubric:    "应用通俗语言解释 MACD 指标含义，不应调用行情或回测工具。",
			MustCover: []string{"MACD"},
		}
	case "chat_signal_quality":
		return ExpectReplySpec{
			Rubric:    "在已做买卖点探测后，应对「信号准吗」做定性说明或限制条件，走闲聊/解释而非再次 probe。",
		}
	case "bot_reminder_list":
		return ExpectReplySpec{
			Rubric:    "应列出或说明用户当前的 DCA reminder，信息结构清晰。",
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
