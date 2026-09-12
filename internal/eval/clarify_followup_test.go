package eval

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestDialogueExecutionPlanSplitsClarifyTurns(t *testing.T) {
	opts := TurnPlanCaseOptions{
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我回测一下中际旭创"},
			{Role: "user", Text: "用SAR加MACD组合回测", OnClarify: true},
		},
	}.Normalize()

	regular, clarify := DialogueExecutionPlan(opts)
	if len(regular) != 1 || regular[0].Text != "帮我回测一下中际旭创" {
		t.Fatalf("regular=%v", regular)
	}
	if len(clarify) != 1 || clarify[0].Text != "用SAR加MACD组合回测" {
		t.Fatalf("clarify=%v", clarify)
	}
}

func TestNeedsClarifyFollowupWhenRequiredToolMissing(t *testing.T) {
	opts := TurnPlanCaseOptions{
		RequireTools: []string{"run_strategy_backtest"},
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我回测一下中际旭创"},
			{Role: "user", Text: "用SAR加MACD组合回测", OnClarify: true},
		},
	}.Normalize()

	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Tools: []string{"search_code"}},
			},
		},
	}
	if !NeedsClarifyFollowup(chat, opts) {
		t.Fatal("expected clarify follow-up")
	}

	chat.Metadata["turn_tools_trace"] = []chatsession.TurnToolsEntry{
		{Tools: []string{"search_code", "run_strategy_backtest"}},
	}
	if NeedsClarifyFollowup(chat, opts) {
		t.Fatal("expected no clarify follow-up once tool ran")
	}
}

func TestPickClarifyAnswerMatchesChoice(t *testing.T) {
	answer, ok := PickClarifyAnswer("请选择策略", []string{
		"SAR信号搭配MACD直方图趋势",
		"MACD金叉死叉",
	}, []string{"用SAR加MACD组合回测"})
	if !ok {
		t.Fatal("expected match")
	}
	if answer != "SAR信号搭配MACD直方图趋势" {
		t.Fatalf("answer=%q", answer)
	}
}

func TestSplitClarifyScriptRunsPostTurnFollowup(t *testing.T) {
	opts := TurnPlanCaseOptions{
		ExpectMode:   "clarify",
		ExpectDomain: "ambiguous",
		ClarifyReply: "先问答，先不操作",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "这个MACD信号平时该怎么用比较好"},
			{Role: "user", Text: "先问答，先不操作", OnClarify: true, Judge: true},
		},
	}.Normalize()

	chat := &chatsession.ChatSession{Metadata: map[string]any{}}
	if !UsesSplitClarifyScript(opts) {
		t.Fatal("expected split clarify script")
	}
	if !NeedsClarifyFollowup(chat, opts) {
		t.Fatal("split clarify cases should run on_clarify follow-up turn")
	}
}

func TestPickClarifyAnswerMatchesChineseChoice(t *testing.T) {
	answer, ok := PickClarifyAnswer("你是想做哪一件？", []string{
		"个股/指标分析", "测买卖点", "跑回测看收益", "先问答，先不操作",
	}, []string{"个股/指标分析"})
	if !ok || answer != "个股/指标分析" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
}

func TestSignalListThenProbeSkipsPostClarifyFollowup(t *testing.T) {
	var opts TurnPlanCaseOptions
	for _, c := range IndividualTurnPlanEvalCases() {
		if c.ID == "turn_plan_signal_list_then_probe" {
			opts = c.Options.Normalize()
			break
		}
	}
	if !HasJudgedRegularTurn(opts) {
		t.Fatal("expected judged regular turn")
	}
	chat := &chatsession.ChatSession{Metadata: map[string]any{}}
	if NeedsClarifyFollowup(chat, opts) {
		t.Fatal("on_clarify turns should stay ClarifyFn-only when judge is in regular dialogue")
	}
	if got := JudgedUserTurnIndex(opts, chat); got != 3 {
		t.Fatalf("judged turn=%d want 3", got)
	}
}

func TestSignalListThenProbeClarifyDefaults(t *testing.T) {
	var opts TurnPlanCaseOptions
	for _, c := range IndividualTurnPlanEvalCases() {
		if c.ID == "turn_plan_signal_list_then_probe" {
			opts = c.Options.Normalize()
			break
		}
	}
	defaults := ClarifyDefaultTexts(opts)
	if len(defaults) != 3 {
		t.Fatalf("defaults=%v want 3", defaults)
	}
	regular, clarify := DialogueExecutionPlan(opts)
	if len(regular) != 3 || len(clarify) != 3 {
		t.Fatalf("regular=%d clarify=%d", len(regular), len(clarify))
	}
	answer, ok := PickClarifyAnswer("为腾讯选哪个组合信号？", []string{
		"SAR信号配套MACD直方图趋势（趋势+动量双确认）",
		"MACD金死叉配套SAR趋势（金死叉更敏感）",
	}, defaults)
	if !ok || answer != "SAR信号配套MACD直方图趋势（趋势+动量双确认）" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
}

func TestClarifyReplyCoverageByCaseKind(t *testing.T) {
	expect := map[string]struct {
		clarifyReply string
		postFollowup bool
	}{
		"turn_plan_backtest_colloquial":    {clarifyReply: "用SAR加MACD组合回测", postFollowup: true},
		"turn_plan_signal_probe_direct":    {clarifyReply: "用SAR加MACD组合测买卖点", postFollowup: true},
		"turn_plan_backtest_explicit":      {clarifyReply: "用默认参数，最近3个月日线", postFollowup: true},
		"turn_plan_dca_grid_backtest":      {clarifyReply: "用默认定投参数回测腾讯控股", postFollowup: true},
		"turn_plan_ambiguous_bare_macd":          {clarifyReply: "SAR信号搭配MACD直方图趋势", postFollowup: true},
		"turn_plan_compound_analysis_backtest": {clarifyReply: "先只做分析", postFollowup: true},
		"turn_plan_stock_quote_ambiguous":        {clarifyReply: "只要当前价", postFollowup: true},
	}
	seen := map[string]bool{}
	for _, c := range IndividualTurnPlanEvalCases() {
		want, ok := expect[c.ID]
		if !ok {
			continue
		}
		seen[c.ID] = true
		if strings.TrimSpace(c.Options.ClarifyReply) != want.clarifyReply {
			t.Fatalf("%s clarify_reply=%q want %q", c.ID, c.Options.ClarifyReply, want.clarifyReply)
		}
		_, clarify := DialogueExecutionPlan(c.Options)
		if want.postFollowup && len(clarify) != 1 {
			t.Fatalf("%s postFollowup clarify turns=%d", c.ID, len(clarify))
		}
		if !want.postFollowup && len(clarify) != 0 {
			t.Fatalf("%s should not have on_clarify dialogue turns", c.ID)
		}
	}
	for id := range expect {
		if !seen[id] {
			t.Fatalf("missing case %s", id)
		}
	}
}
