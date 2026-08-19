package notify

import (
	"fmt"
	"strings"
)

// FormatTradeMarkdown renders trading notices for Feishu IM (feishupush markdown post).
// Category titles and field labels are mirrored by GeeGooBot internal/notify/push_text.go (JPush).
func FormatTradeMarkdown(noticeType int, content map[string]any) string {
	switch noticeType {
	case 0:
		return "#### 叽咕助手通知绑定成功\n\n叽咕助手上线了！"
	case 1:
		return formatOrderMarkdown(content)
	case 2:
		return formatSignalMarkdown(content)
	case 3:
		return formatGridMarkdown(content)
	case 4:
		return formatBotOptMarkdown(content)
	case 5:
		return formatReminderMarkdown(content)
	default:
		return fmt.Sprintf("#### 叽咕助手通知\n\n```json\n%v\n```", content)
	}
}

func formatOrderMarkdown(content map[string]any) string {
	lines := []string{
		"#### 实时通知-订单-(" + tradeEnv(content) + ")",
		"",
		"- **用户名:** " + str(content["user"]),
		"- **名称:** " + str(content["botname"]),
		"- **类型:** " + botTypeLabel(str(content["botType"])),
		"- **操作股票:** " + str(content["stock_name"]) + "[" + str(content["code"]) + "]",
		"- **操作类型:** " + orderTypeLabel(str(content["order_type"])),
		"- **操作价格:** " + str(content["price"]),
		"- **操作数量:** " + str(content["qty"]),
	}
	return strings.Join(lines, "\n")
}

func formatSignalMarkdown(content map[string]any) string {
	lines := []string{
		"#### 实时通知-信号触发-(" + tradeEnv(content) + ")",
		"",
		"- **用户名:** " + str(content["user"]),
		"- **名称:** " + str(content["botname"]),
		"- **类型:** " + botTypeLabel(str(content["botType"])),
		"- **操作股票:** " + str(content["stock_name"]) + "[" + str(content["code"]) + "]",
		"- **买入信号:** " + str(content["buy_signal"]),
		"- **卖出信号:** " + sellSignalLabel(content["sell_signal"]),
		"- **建议操作:** " + optSignalLabel(str(content["next_opt"])),
	}
	if ta := tradeAgentMarkdown(content["trade_agent"]); ta != "" {
		lines = append(lines, "", ta)
	}
	return strings.Join(lines, "\n")
}

func formatGridMarkdown(content map[string]any) string {
	lines := []string{
		"#### 实时通知-网格触发-(" + tradeEnv(content) + ")",
		"",
		"- **用户名:** " + str(content["user"]),
		"- **名称:** " + str(content["botname"]),
		"- **类型:** " + botTypeLabel(str(content["botType"])),
		"- **操作股票:** " + str(content["stock_name"]) + "[" + str(content["code"]) + "]",
		"- **突破网格:** " + str(content["break_grid"]),
		"- **建议操作:** " + optSignalLabel(str(content["next_opt"])),
	}
	if ta := tradeAgentMarkdown(content["trade_agent"]); ta != "" {
		lines = append(lines, "", ta)
	}
	return strings.Join(lines, "\n")
}

func formatBotOptMarkdown(content map[string]any) string {
	lines := []string{
		"#### 机器人操作",
		"",
		"- **用户名:** " + str(content["user"]),
		"- **名称:** " + str(content["botname"]),
		"- **类型:** " + botTypeLabel(str(content["botType"])),
		"- **股票:** " + str(content["stock_name"]) + "[" + str(content["code"]) + "]",
		"- **操作:** " + botOptLabel(str(content["opt"])),
	}
	return strings.Join(lines, "\n")
}

func formatReminderMarkdown(content map[string]any) string {
	lines := []string{
		"#### 实时通知-止盈止损提醒-(" + tradeEnv(content) + ")",
		"",
		"- **用户名:** " + str(content["user"]),
		"- **名称:** " + str(content["botname"]),
		"- **类型:** " + botTypeLabel(str(content["botType"])),
		"- **操作股票:** " + str(content["stock_name"]) + "[" + str(content["code"]) + "]",
		"- **当前价格:** " + str(content["current_price"]),
		"- **成本价格:** " + str(content["cost_price"]),
		"- **提醒类型:** " + reminderOptLabel(str(content["next_opt"])),
		"- **盈亏比例:** " + str(content["pl_ratio"]),
	}
	return strings.Join(lines, "\n")
}

func tradeAgentMarkdown(v any) string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	result := strings.ToLower(str(m["result"]))
	conf := strings.ToLower(str(m["confidence"]))
	decision := result
	if decision != "buy" && decision != "sell" && decision != "hold" {
		if decision == "" {
			decision = "-"
		}
	}
	confDisp := conf
	if confDisp != "high" && confDisp != "medium" && confDisp != "low" {
		if confDisp == "" {
			confDisp = "-"
		}
	}
	return "**智能体决策:** " + decision + "\n**置信度:** " + confDisp
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func tradeEnv(content map[string]any) string {
	switch strings.ToUpper(str(content["trade_env"])) {
	case "REAL":
		return "真实环境"
	case "SIMULATE":
		return "模拟环境"
	default:
		return str(content["trade_env"])
	}
}

func botTypeLabel(t string) string {
	switch t {
	case "DCA":
		return "信号交易机器人"
	case "SmartTrade":
		return "智能交易机器人"
	case "GRID":
		return "网格交易机器人"
	case "DCAReminder":
		return "信号提醒机器人"
	case "GRIDReminder":
		return "网格提醒机器人"
	case "SmartReminder":
		return "智能交易提醒机器人"
	case "HDG":
		return "对冲交易机器人"
	default:
		return t
	}
}

func orderTypeLabel(opt string) string {
	switch opt {
	case "hold":
		return "无操作"
	case "buy":
		return "手动买入"
	case "smart_buy":
		return "智能买入"
	case "smart_sell":
		return "智能卖出"
	case "signal_buy":
		return "信号买入"
	case "signal_sell":
		return "信号卖出"
	case "grid_buy":
		return "网格买入"
	case "grid_sell":
		return "网格卖出"
	case "profit_take_tailing_sell":
		return "追踪止盈卖出"
	case "profit_take_sell":
		return "止盈卖出"
	case "stop_loss_tailing_sell":
		return "追踪止损卖出"
	case "stop_loss_sell":
		return "止损卖出"
	default:
		return opt
	}
}

func optSignalLabel(opt string) string {
	switch opt {
	case "buy":
		return "买入"
	case "sell":
		return "卖出"
	case "hold":
		return "持有"
	default:
		return opt
	}
}

func sellSignalLabel(v any) string {
	s := str(v)
	if strings.Contains(s, "NOSIGNAL") {
		return "未开启"
	}
	return s
}

func botOptLabel(opt string) string {
	switch opt {
	case "create":
		return "创建"
	case "edit":
		return "修改"
	case "delete":
		return "删除"
	default:
		return opt
	}
}

func reminderOptLabel(opt string) string {
	switch opt {
	case "tp":
		return "止盈"
	case "sl":
		return "止损"
	case "tp_trailing_start":
		return "止盈跟踪启动"
	case "sl_trailing_start":
		return "止损跟踪启动"
	case "tp_trailing":
		return "止盈跟踪触发"
	case "sl_trailing":
		return "止损跟踪触发"
	case "hold":
		return "持有"
	default:
		return opt
	}
}
