package eval

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// TurnPlanCaseOptions is one dashboard eval case (live chat execution).
type TurnPlanCaseOptions struct {
	Category       string   `json:"category"`
	PlanOnly       bool     `json:"plan_only"`
	SessionCleanup string   `json:"session_cleanup"`
	DualModelEval  bool     `json:"dual_model_eval"`
	Message        string   `json:"message"`
	SetupMessages  []string `json:"setup_messages,omitempty"`
	ExpectDomain   string   `json:"expect_domain"`
	ExpectMode     string   `json:"expect_mode"`
	ExpectSOP      bool     `json:"expect_sop"`
	ForbidTools    []string `json:"forbid_tools,omitempty"`
	RequireTools   []string `json:"require_tools,omitempty"`
	PassKeywords   []string `json:"pass_keywords,omitempty"`
	MinReplyChars  int      `json:"min_reply_chars,omitempty"`
	TurnID         string   `json:"turn_id,omitempty"`
}

// TurnPlanEvalCaseDef is one seeded dashboard eval row.
type TurnPlanEvalCaseDef struct {
	ID          string
	Title       string
	Description string
	Steps       []string
	SortOrder   int
	Options     TurnPlanCaseOptions
}

// ParseTurnPlanCaseOptions decodes options_json for a live turn_plan eval case.
func ParseTurnPlanCaseOptions(raw []byte) (TurnPlanCaseOptions, error) {
	var opts TurnPlanCaseOptions
	if err := json.Unmarshal(raw, &opts); err != nil {
		return TurnPlanCaseOptions{}, err
	}
	if opts.Category == "" {
		opts.Category = "turn_plan"
	}
	return opts, nil
}

// IndividualTurnPlanEvalCases expands the canonical suite into separate dashboard cases.
func IndividualTurnPlanEvalCases() []TurnPlanEvalCaseDef {
	turns := defaultTurnPlanTurns()
	out := make([]TurnPlanEvalCaseDef, 0, len(turns))
	for i, turn := range turns {
		opts := TurnPlanCaseOptions{
			Category:       "turn_plan",
			PlanOnly:       false,
			SessionCleanup: "before_run",
			DualModelEval:  false,
			Message:        turn.Message,
			SetupMessages:  setupMessagesForTurn(turn),
			ExpectDomain:   turn.ExpectDomain,
			ExpectMode:     turn.ExpectMode,
			ExpectSOP:      turn.ExpectSOP,
			ForbidTools:    append([]string(nil), turn.ForbidTools...),
			RequireTools:   append([]string(nil), turn.RequireTools...),
			MinReplyChars:  20,
			TurnID:         turn.ID,
		}
		id := "turn_plan_" + turn.ID
		title := "TurnPlan · " + turnPlanTitle(turn.ID)
		desc := fmt.Sprintf("真实 Chat 执行：发送「%s」，校验 turn_plan 路由与工具调用。", turn.Message)
		steps := []string{
			"清空 Dock Chat 会话",
			"按 session 话术发送用户消息（含必要的 setup 轮次）",
			"校验 turn_plan domain/mode/SOP 与实际工具调用",
		}
		if len(opts.SetupMessages) > 0 {
			steps[1] = "先发送 setup 话术建立上下文，再发送目标用户消息"
		}
		out = append(out, TurnPlanEvalCaseDef{
			ID:          id,
			Title:       title,
			Description: desc,
			Steps:       steps,
			SortOrder:   50 + i,
			Options:     opts,
		})
	}
	return out
}

func setupMessagesForTurn(turn TurnPlanTurn) []string {
	switch turn.ID {
	case "stock_followup_technical":
		return []string{"帮我查一下腾讯的股价"}
	case "sticky_symbol_switch":
		return []string{"帮我分析一下中际旭创"}
	case "backtest_after_analysis":
		return []string{"帮我分析一下小米"}
	default:
		if turn.LastDomain != "" {
			return []string{"帮我分析一下中际旭创"}
		}
		return nil
	}
}

func turnPlanTitle(id string) string {
	titles := map[string]string{
		"stock_price_lookup":         "查腾讯股价",
		"stock_followup_technical":   "分析技术面跟进",
		"stock_colloquial":           "口语换股票",
		"signal_probe":               "测买卖点",
		"signal_probe_combo":         "SAR+MACD 测点",
		"backtest_explicit":          "显式回测",
		"backtest_colloquial":        "口语回测",
		"ambiguous_bare_macd":        "模糊 MACD",
		"compound_analysis_backtest": "分析+回测复合",
		"chat_definition":            "MACD 释义",
		"chat_signal_quality":        "信号准吗",
		"bot_reminder_list":          "Reminder 列表",
		"bot_grid_pnl":               "网格 Bot 盈亏",
		"bot_smarttrade_list":        "SmartTrade 列表",
		"backtest_history":           "回测历史",
		"report_lookup":              "盘前报告",
		"knowledge_lookup":           "知识库检索",
		"news_lookup":                "新闻查询",
		"dca_grid_backtest":          "DCA 回测",
		"sticky_symbol_switch":       "换贵州茅台",
		"backtest_after_analysis":    "分析后回测",
	}
	if t, ok := titles[id]; ok {
		return t
	}
	return id
}

// TurnPlanSnapshot is persisted on the chat session after each agent turn.
type TurnPlanSnapshot struct {
	Domain     string   `json:"domain"`
	Mode       string   `json:"mode"`
	SOP        bool     `json:"sop"`
	ToolsAllow []string `json:"tools_allow,omitempty"`
}

// VerifyTurnPlanLive checks a completed chat turn against case expectations.
func VerifyTurnPlanLive(chat *chatsession.ChatSession, opts TurnPlanCaseOptions) TurnPlanResult {
	turnID := opts.TurnID
	if turnID == "" {
		turnID = "live"
	}
	res := TurnPlanResult{TurnID: turnID, Passed: true}
	if chat == nil {
		res.Passed = false
		res.Detail = "session not found"
		return res
	}

	snap, ok := chatsession.LastTurnPlanFromSession(chat)
	if !ok {
		res.Passed = false
		res.Detail = "missing last_turn_plan on session (turn did not complete?)"
		return res
	}

	var problems []string
	if snap.Domain != opts.ExpectDomain {
		problems = append(problems, fmt.Sprintf("domain=%s want %s", snap.Domain, opts.ExpectDomain))
	}
	if snap.Mode != opts.ExpectMode {
		problems = append(problems, fmt.Sprintf("mode=%s want %s", snap.Mode, opts.ExpectMode))
	}
	if snap.SOP != opts.ExpectSOP {
		problems = append(problems, fmt.Sprintf("sop=%v want %v", snap.SOP, opts.ExpectSOP))
	}

	called := chatsession.LastTurnToolsCalledFromSession(chat)
	for _, tool := range opts.ForbidTools {
		if containsString(called, tool) {
			problems = append(problems, fmt.Sprintf("forbid tool %s called", tool))
		}
	}
	for _, tool := range opts.RequireTools {
		if !containsString(called, tool) {
			problems = append(problems, fmt.Sprintf("missing tool call %s", tool))
		}
	}

	if len(problems) > 0 {
		res.Passed = false
		res.Detail = strings.Join(problems, "; ")
		return res
	}
	res.Detail = fmt.Sprintf("%s/%s sop=%v tools=%s", snap.Domain, snap.Mode, snap.SOP, strings.Join(called, ","))
	return res
}

func containsString(list []string, name string) bool {
	for _, t := range list {
		if t == name {
			return true
		}
	}
	return false
}
