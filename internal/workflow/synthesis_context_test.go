package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type contextCheckingSynthesizer struct {
	key  any
	want any
	seen any
}

func (c *contextCheckingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	c.seen = ctx.Value(c.key)
	return strings.Repeat("context-aware synthesis ", 8), "hold", "context summary", nil
}

func TestBuildCreateReportArgsPassesContextToSynthesis(t *testing.T) {
	key := struct{}{}
	synth := &contextCheckingSynthesizer{key: key, want: "run-context"}
	workflow.SetDefaultSynthesizer(synth)
	defer workflow.SetDefaultSynthesizer(nil)

	w := memory.NewPreMarketWorking("s3", "pre_market")
	w.BotCodes = []memory.BotStock{{Code: "00700.HK", StockName: "Tencent", BotID: "b1", BotName: "bot", BotType: "DCA"}}
	w.Stocks["00700.HK"] = memory.StockWorkspace{Code: "00700.HK", StockName: "Tencent", BotID: "b1", BotName: "bot", BotType: "DCA", Attitude: "neutral"}

	ctx := context.WithValue(context.Background(), key, synth.want)
	args := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if synth.seen != synth.want {
		t.Fatalf("synthesis did not receive workflow context: got %v want %v", synth.seen, synth.want)
	}
	if args["summary"] != "context summary" {
		t.Fatalf("summary should come from context-aware synthesis, got %v", args["summary"])
	}
}
