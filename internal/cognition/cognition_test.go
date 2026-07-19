package cognition_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func TestIdentityRankerPreservesOrder(t *testing.T) {
	t.Parallel()
	items := []cognition.RankItem{
		{ID: "a", Text: "first", Score: 0.1},
		{ID: "b", Text: "second", Score: 0.9},
	}
	out, err := cognition.IdentityRanker{}.Rank(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].ID != "a" || out[1].ID != "b" {
		t.Fatalf("got %+v", out)
	}
}

func TestAcceptAllEvaluatorAccepts(t *testing.T) {
	t.Parallel()
	res, err := cognition.AcceptAllEvaluator{}.Evaluate(context.Background(), cognition.EvalInput{
		AssistantText: "ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accept || res.RetrySuggested {
		t.Fatalf("got %+v", res)
	}
}

func TestDefaultPlanPolicyHoldAndApproval(t *testing.T) {
	t.Parallel()
	p := cognition.DefaultPlanPolicy{}

	if !p.ShouldHold(cognition.PlanHoldInput{
		GateEnabled: true, Interactive: true, Approved: false, MutatingCount: 1,
	}) {
		t.Fatal("expected hold")
	}
	if p.ShouldHold(cognition.PlanHoldInput{
		GateEnabled: true, Interactive: true, Approved: true, MutatingCount: 1,
	}) {
		t.Fatal("approved should not hold")
	}
	if !p.IsApproval("y") || !p.IsApproval("确认") {
		t.Fatal("approval tokens")
	}
	if !p.IsRejection("n") || !p.IsRejection("取消") {
		t.Fatal("rejection tokens")
	}
	msg := p.HoldMessage("")
	if msg == "" || !strings.Contains(msg, "写操作待确认") {
		t.Fatalf("hold message=%q", msg)
	}
	payload := p.ProposedPayload([]cognition.ProposedCall{
		{Name: "create_dca_bot", Arguments: map[string]any{"x": 1}},
	})
	tools, _ := payload["tools"].([]string)
	if len(tools) != 1 || tools[0] != "create_dca_bot" {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestDefaultsBundle(t *testing.T) {
	t.Parallel()
	d := cognition.Defaults()
	if d.Ranker == nil || d.Evaluator == nil || d.PlanPolicy == nil {
		t.Fatalf("defaults incomplete: %+v", d)
	}
}
