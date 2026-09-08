package eval_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type stubJudge struct {
	out eval.ReplyJudgeOutput
}

func (s stubJudge) Judge(_ context.Context, _ eval.ReplyJudgeInput) (eval.ReplyJudgeOutput, error) {
	return s.out, nil
}

func TestVerifyTurnPlanLiveFullWithJudge(t *testing.T) {
	chat := &chatsession.ChatSession{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "帮我查一下腾讯的股价"},
			{Role: llm.RoleAssistant, Content: "腾讯控股现价 320 港元，今日小幅上涨。"},
		},
		Metadata: map[string]any{
			"last_turn_plan": map[string]any{
				"domain": "stock_analysis", "mode": "gather", "sop": true,
			},
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_current_price"}},
			},
		},
	}
	opts := eval.TurnPlanCaseOptions{
		TurnID: "stock_price", ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
		Message: "帮我查一下腾讯的股价",
		ExecutionProfile: "stock_analysis.price_snapshot",
		MinReplyChars: 10,
	}.Normalize()

	res := eval.VerifyTurnPlanLiveFull(context.Background(), chat, opts, stubJudge{
		out: eval.ReplyJudgeOutput{Pass: true, Score: 0.9, Reason: "报价合理"},
	})
	if !res.Passed {
		t.Fatalf("expected pass, got %s", res.Detail)
	}
	if res.ActualReply == "" {
		t.Fatal("missing actual reply")
	}
	foundIntent := false
	foundExecution := false
	foundJudge := false
	for _, c := range res.Checks {
		switch c.Type {
		case "intent":
			foundIntent = true
		case "execution":
			foundExecution = true
		case "llm_judge":
			foundJudge = true
		}
	}
	if !foundIntent {
		t.Fatal("missing intent check")
	}
	if !foundExecution {
		t.Fatal("missing execution check")
	}
	if !foundJudge {
		t.Fatal("missing llm_judge check")
	}
}

func TestVerifyTurnPlanLiveFullJudgeFail(t *testing.T) {
	chat := &chatsession.ChatSession{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "这个MACD信号怎么弄比较好"},
			{Role: llm.RoleAssistant, Content: "已开始为您回测。"},
		},
		Metadata: map[string]any{
			"last_turn_plan": map[string]any{
				"domain": "ambiguous", "mode": "clarify", "sop": false,
			},
			"last_turn_tools_called": []any{},
		},
	}
	opts := eval.TurnPlanCaseOptions{
		TurnID: "ambiguous_bare_macd", ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
		Message: "这个MACD信号怎么弄比较好",
	}.Normalize()

	res := eval.VerifyTurnPlanLiveFull(context.Background(), chat, opts, stubJudge{
		out: eval.ReplyJudgeOutput{Pass: false, Score: 0.2, Reason: "误执行回测"},
	})
	if res.Passed {
		t.Fatal("expected fail when judge fails")
	}
}
