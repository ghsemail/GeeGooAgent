package workflow_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type contextCheckingSynthesizer struct {
	key  any
	want string
	got  context.Context
}

func (c *contextCheckingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	c.got = ctx
	return "reason " + stringRepeat("x", 80), "hold", "ok", nil
}

func stringRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestBuildCreateReportArgsPassesContextToSynthesis(t *testing.T) {
	key, want := "run-context", "run-context"
	ctx := context.WithValue(context.Background(), key, want)
	ctx = workflow.ContextWithSynthesizer(ctx, &contextCheckingSynthesizer{key: key, want: want})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK"}},
		Stocks:   map[string]memory.StockWorkspace{"00700.HK": {Code: "00700.HK", Attitude: "neutral"}},
	}
	args := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if args["suggestion"] != "hold" {
		t.Fatalf("args=%v", args)
	}
}
