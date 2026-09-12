package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SuggestedCaseExpectations is the auto-generated配套 for a TurnPlan live case.
type SuggestedCaseExpectations struct {
	Title             string              `json:"title"`
	Description       string              `json:"description"`
	Steps             []string            `json:"steps"`
	ExpectIntent      ExpectIntentSpec    `json:"expect_intent"`
	ExpectRouting     ExpectRoutingSpec   `json:"expect_routing"`
	ExpectReply       ExpectReplySpec     `json:"expect_reply"`
	ExpectExecution   *ExpectExecutionSpec `json:"expect_execution,omitempty"`
	Judge             *EvalJudgeConfig    `json:"judge,omitempty"`
	PlannerLastDomain string              `json:"planner_last_domain,omitempty"`
}

// ExpectReplySuggester optionally uses an LLM to draft rubric fields.
type ExpectReplySuggester interface {
	Suggest(ctx context.Context, dialogue []EvalDialogueTurn, intent ExpectIntentSpec, requireTools []string) (ExpectReplySpec, error)
}

// LLMExpectReplySuggester drafts rubric / must_cover / must_not from dialogue + routing.
type LLMExpectReplySuggester struct {
	Provider llm.Provider
	Policy   llm.Policy
}

func (g *LLMExpectReplySuggester) Suggest(
	ctx context.Context,
	dialogue []EvalDialogueTurn,
	intent ExpectIntentSpec,
	requireTools []string,
) (ExpectReplySpec, error) {
	if g == nil || g.Provider == nil {
		return heuristicExpectReply(dialogue, intent), nil
	}
	temp := 0.2
	maxTokens := 512
	if g.Policy != nil {
		dec := g.Policy.Decide(llm.Request{Kind: llm.TaskSynthesis})
		if dec.Temperature > 0 {
			temp = dec.Temperature
		}
		if dec.MaxTokens > 0 && dec.MaxTokens < maxTokens*4 {
			maxTokens = dec.MaxTokens
		}
	}
	judgeText, contextLines := dialogueJudgeContext(dialogue)
	system := `你是 GeeGoo Agent Eval 用例编辑助手。根据多轮用户剧本与期望路由，生成 Live Eval 的验收标准。
只输出 JSON：{"rubric":"…","must_cover":["…"],"must_not":["…"]}
- rubric 描述评判轮助手应达成的业务意图（1-2 句）
- must_cover 2-4 个应出现在回复中的关键词（标的/动作）
- must_not 0-3 个不应出现的误导性动作（如误跑回测、误 probe）`
	user := fmt.Sprintf(`## 对话剧本
%s

## 评判轮用户话术
%s

## 期望路由
domain=%s mode=%s act=%s

## 期望工具
%s`,
		strings.Join(contextLines, "\n"),
		judgeText,
		intent.Domain, intent.Mode, intent.Act,
		strings.Join(requireTools, ", "),
	)
	resp, err := g.Provider.Chat(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: user},
	}, nil, temp, maxTokens)
	if err != nil {
		return heuristicExpectReply(dialogue, intent), err
	}
	spec, err := parseExpectReplyJSON(resp.Content)
	if err != nil {
		return heuristicExpectReply(dialogue, intent), err
	}
	if strings.TrimSpace(spec.Rubric) == "" {
		return heuristicExpectReply(dialogue, intent), fmt.Errorf("empty rubric from LLM")
	}
	return spec, nil
}

func parseExpectReplyJSON(raw string) (ExpectReplySpec, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var out ExpectReplySpec
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return ExpectReplySpec{}, err
	}
	return out, nil
}

// SuggestCaseExpectations derives routing/reply/metadata from dialogue using the production planner.
func SuggestCaseExpectations(
	ctx context.Context,
	dialogue []EvalDialogueTurn,
	planner cognition.Planner,
	replySuggester ExpectReplySuggester,
) (SuggestedCaseExpectations, error) {
	turns := normalizeDialogueForSuggest(dialogue)
	if len(turns) == 0 {
		return SuggestedCaseExpectations{}, fmt.Errorf("dialogue required")
	}
	judgeIdx := -1
	for i, t := range turns {
		if t.Judge {
			judgeIdx = i
			break
		}
	}
	if judgeIdx < 0 {
		judgeIdx = len(turns) - 1
		turns[judgeIdx].Judge = true
	}

	lastDomain := ""
	if planner != nil {
		for i := 0; i < judgeIdx; i++ {
			text := strings.TrimSpace(turns[i].Text)
			if text == "" {
				continue
			}
			plan := planner.Plan(cognition.PlanInput{
				Ctx:        ctx,
				UserText:   text,
				LastDomain: cognition.Domain(lastDomain),
			})
			lastDomain = string(plan.Domain)
		}
	}

	judgeText := strings.TrimSpace(turns[judgeIdx].Text)
	var plan cognition.TurnPlan
	if planner != nil {
		plan = planner.Plan(cognition.PlanInput{
			Ctx:        ctx,
			UserText:   judgeText,
			LastDomain: cognition.Domain(lastDomain),
		})
	} else {
		plan = cognition.TurnPlan{Domain: cognition.DomainBacktestRun, Mode: cognition.ModeExecute, Act: "backtest"}
	}

	intent := ExpectIntentSpec{
		Domain: string(plan.Domain),
		Mode:   string(plan.Mode),
		SOP:    false,
		Act:    strings.TrimSpace(plan.Act),
	}
	requireTools, forbidTools := legacyToolsForIntent(intent)
	routing := ExpectRoutingSpec{
		Domain:       intent.Domain,
		Mode:         intent.Mode,
		SOP:          intent.SOP,
		RequireTools: requireTools,
		ForbidTools:  forbidTools,
	}

	var reply ExpectReplySpec
	var replyErr error
	if replySuggester != nil {
		reply, replyErr = replySuggester.Suggest(ctx, turns, intent, requireTools)
	} else {
		reply = heuristicExpectReply(turns, intent)
	}
	if strings.TrimSpace(reply.Rubric) == "" {
		reply = heuristicExpectReply(turns, intent)
	}

	setup, message := utterancesFromDialogueTurns(turns)
	live := TurnPlanLiveCase{
		Title:         suggestTitle(turns, intent),
		Description:   suggestDescription(turns, intent),
		SetupMessages: setup,
		Message:       message,
		ExpectDomain:  intent.Domain,
		ExpectMode:    intent.Mode,
		ExpectAct:     intent.Act,
		ExpectSOP:     intent.SOP,
		RequireTools:  requireTools,
		ForbidTools:   forbidTools,
	}
	steps := liveCaseSteps(live)

	var exec *ExpectExecutionSpec
	if len(requireTools) > 0 || len(forbidTools) > 0 {
		exec = &ExpectExecutionSpec{
			LegacyRequireTools: append([]string(nil), requireTools...),
			ForbidTools:        append([]string(nil), forbidTools...),
		}
	}

	out := SuggestedCaseExpectations{
		Title:             "TurnPlan · " + live.Title,
		Description:       live.Description,
		Steps:             steps,
		ExpectIntent:      intent,
		ExpectRouting:     routing,
		ExpectReply:       reply,
		ExpectExecution:   exec,
		Judge:             &EvalJudgeConfig{Enabled: true, ModelSlot: "auxiliary", MinScore: 0.7},
		PlannerLastDomain: lastDomain,
	}
	return out, replyErr
}

func normalizeDialogueForSuggest(dialogue []EvalDialogueTurn) []EvalDialogueTurn {
	out := make([]EvalDialogueTurn, 0, len(dialogue))
	for _, t := range dialogue {
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		out = append(out, EvalDialogueTurn{
			Role:      t.Role,
			Text:      text,
			Judge:     t.Judge,
			OnClarify: t.OnClarify,
		})
	}
	return out
}

func utterancesFromDialogueTurns(turns []EvalDialogueTurn) (setup []string, message string) {
	if len(turns) == 0 {
		return nil, ""
	}
	judgeIdx := len(turns) - 1
	for i, t := range turns {
		if t.Judge {
			judgeIdx = i
			break
		}
	}
	for i := 0; i < judgeIdx; i++ {
		setup = append(setup, turns[i].Text)
	}
	return setup, turns[judgeIdx].Text
}

func dialogueJudgeContext(dialogue []EvalDialogueTurn) (judgeText string, lines []string) {
	turns := normalizeDialogueForSuggest(dialogue)
	for i, t := range turns {
		prefix := fmt.Sprintf("%d. ", i+1)
		if t.Judge {
			prefix = fmt.Sprintf("%d. [评判] ", i+1)
			judgeText = t.Text
		}
		lines = append(lines, prefix+t.Text)
	}
	if judgeText == "" && len(turns) > 0 {
		judgeText = turns[len(turns)-1].Text
	}
	return judgeText, lines
}

func legacyToolsForIntent(intent ExpectIntentSpec) (require, forbid []string) {
	domain := strings.TrimSpace(intent.Domain)
	mode := strings.TrimSpace(intent.Mode)
	switch domain {
	case "backtest_run":
		return []string{"run_strategy_backtest"}, []string{"probe_bot_signal_series"}
	case "signal_probe":
		return []string{"probe_bot_signal_series"}, []string{"run_strategy_backtest"}
	case "dca_grid":
		if mode == "gather" {
			return []string{"get_signal_combinations"}, []string{"run_strategy_backtest", "probe_bot_signal_series"}
		}
		return []string{"generate_dca_strategy"}, nil
	case "stock_analysis":
		if intent.Act == "quote_price" {
			return []string{"search_code", "get_current_price"}, []string{"run_strategy_backtest"}
		}
		return []string{"search_code", "get_mcp_analysis"}, []string{"run_strategy_backtest"}
	case "backtest_history":
		return []string{"list_strategy_backtest_logs"}, nil
	case "bot_manage":
		return []string{"list_dca_reminders"}, nil
	case "report_lookup":
		return []string{"get_stock_premarket_reports"}, nil
	case "knowledge":
		return []string{"search_knowledge"}, nil
	case "news":
		return []string{"fetch_market_news"}, nil
	case "chat":
		return nil, []string{"run_strategy_backtest", "probe_bot_signal_series"}
	default:
		return nil, nil
	}
}

func heuristicExpectReply(dialogue []EvalDialogueTurn, intent ExpectIntentSpec) ExpectReplySpec {
	_, lines := dialogueJudgeContext(dialogue)
	allText := strings.Join(lines, " ")
	spec := ExpectReplySpec{Rubric: heuristicRubric(intent, allText)}
	spec.MustCover = extractMustCoverKeywords(allText, intent)
	spec.MustNot = heuristicMustNot(intent)
	return spec
}

func heuristicRubric(intent ExpectIntentSpec, dialogueText string) string {
	switch intent.Domain {
	case "backtest_run":
		return "应识别回测意图并针对对话中的标的发起或汇报回测进度/结果，而非只做静态分析或 probe 买卖点。"
	case "signal_probe":
		return "应对指定标的执行或汇报买卖点探测结果，包含信号方向或暂无信号说明，不应直接跑完整回测。"
	case "dca_grid":
		if intent.Mode == "gather" {
			return "应列出或摘要用户可用的信号/组合策略，语气自然，不应直接跑回测或 probe。"
		}
		return "应识别 DCA/网格类方案意图并给出可执行下一步，而非误跑 SmartTrade 回测。"
	case "stock_analysis":
		return "应针对用户关心的标的给出分析或行情信息，符合当前轮意图（报价/走势/技术面等）。"
	case "chat":
		return "应以解释/问答方式回应，不应误触发交易、回测或 probe 工具。"
	default:
		return fmt.Sprintf("最后一轮用户意图应路由到 %s/%s，助手回复需符合该能力域。", intent.Domain, intent.Mode)
	}
}

func heuristicMustNot(intent ExpectIntentSpec) []string {
	switch intent.Domain {
	case "backtest_run":
		return []string{"未找到标的"}
	case "signal_probe":
		return []string{"开始回测"}
	case "stock_analysis":
		return []string{"开始回测"}
	default:
		return nil
	}
}

var stockKeywordHints = []string{
	"腾讯", "茅台", "贵州茅台", "小米", "中际旭创", "中际", "苹果", "AAPL", "TSLA", "特斯拉",
}

func extractMustCoverKeywords(dialogueText string, intent ExpectIntentSpec) []string {
	var out []string
	seen := map[string]bool{}
	for _, kw := range stockKeywordHints {
		if strings.Contains(dialogueText, kw) && !seen[kw] {
			out = append(out, kw)
			seen[kw] = true
		}
	}
	switch intent.Domain {
	case "backtest_run":
		if !seen["回测"] {
			out = append(out, "回测")
		}
	case "signal_probe":
		if !seen["信号"] && !strings.Contains(strings.Join(out, ""), "中际") {
			// keep stock-only
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

func suggestTitle(turns []EvalDialogueTurn, intent ExpectIntentSpec) string {
	n := len(turns)
	if n <= 1 {
		return "单轮 · " + shortIntentLabel(intent)
	}
	return fmt.Sprintf("多轮 · %s", shortIntentLabel(intent))
}

func suggestDescription(turns []EvalDialogueTurn, intent ExpectIntentSpec) string {
	if len(turns) <= 1 {
		return fmt.Sprintf("独立 session：%s", strings.TrimSpace(turns[0].Text))
	}
	parts := make([]string, 0, len(turns))
	for _, t := range turns {
		parts = append(parts, t.Text)
	}
	return fmt.Sprintf("同 session 按序：%s → … → %s。", parts[0], parts[len(parts)-1])
}

func shortIntentLabel(intent ExpectIntentSpec) string {
	switch intent.Domain {
	case "backtest_run":
		return "策略回测"
	case "signal_probe":
		return "测买卖点"
	case "stock_analysis":
		return "股票分析"
	case "dca_grid":
		if intent.Mode == "gather" {
			return "列策略"
		}
		return "DCA/网格"
	case "chat":
		return "闲聊/释义"
	default:
		if intent.Domain != "" {
			return intent.Domain
		}
		return "Live Eval"
	}
}
