package workflow_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type recordingSynthesizer struct {
	called bool
}

func (r *recordingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	r.called = true
	return "", "", "", context.Canceled
}

func TestRunnerInjectsSynthesizerIntoContext(t *testing.T) {
	rec := &recordingSynthesizer{}
	store := infra.NewStateStore(t.TempDir())
	workingStore := memory.NewWorkingStore(store)
	working, err := workingStore.Create("sess-synth", "pre_market")
	if err != nil {
		t.Fatal(err)
	}
	trading := true
	working.IsTradingDay = &trading
	working.BotCodes = []memory.BotStock{{Code: "00700.HK", StockName: "腾讯"}}
	working.Stocks = map[string]memory.StockWorkspace{
		"00700.HK": {Code: "00700.HK", StockName: "腾讯", Status: "collecting", Attitude: "bullish"},
	}
	working.CurrentStock = "00700.HK"

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "create_pre_market_report",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			if workflow.SynthesizerFrom(ctx.GoContext()) == nil {
				t.Fatal("synthesizer not injected into tool context")
			}
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	for _, name := range []string{"noop", "write_execution_log"} {
		n := name
		registry.Register(tools.Tool{
			Name: n,
			Handle: func(tools.Context, map[string]any) tools.Result {
				return tools.Result{Status: tools.StatusOK, Summary: "ok"}
			},
		})
	}

	runner := workflow.NewRunner(runtime.NewExecutor(registry), workingStore, workflow.CheckpointAdapter{})
	runner.SetSynthesizer(rec)

	steps := []workflow.Step{{
		Name: "create_pre_market_report", Tool: "create_pre_market_report",
		ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			if workflow.SynthesizerFrom(ctx) != rec {
				t.Fatal("step args ctx missing synthesizer")
			}
			return workflow.BuildCreateReportArgsContext(ctx, w, w.CurrentStock)
		},
	}}

	result := runner.Run("sess-synth", "pre_market", nil, steps, tools.Context{SessionID: "sess-synth"}, working)
	if result.Status == "failed" && result.LastError != "" {
		t.Fatalf("run failed: %s", result.LastError)
	}
	if !rec.called {
		t.Fatal("expected synthesizer to be invoked during report build")
	}
}
