package workflow_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type failingCheckpointSaver struct{}

func (failingCheckpointSaver) Save(string, string, string, string, int, *memory.PreMarketWorking) error {
	return errors.New("checkpoint failed")
}

func TestRunnerFailsWhenCheckpointSaveFails(t *testing.T) {
	store := infra.NewStateStore(t.TempDir())
	workingStore := memory.NewWorkingStore(store)
	working, err := workingStore.Create("session-1", "pre_market")
	if err != nil {
		t.Fatal(err)
	}

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "noop",
		Handle: func(tools.Context, map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	registry.Register(tools.Tool{
		Name: "write_execution_log",
		Handle: func(tools.Context, map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"path": "log.json"}}
		},
	})

	runner := workflow.NewRunner(runtime.NewExecutor(registry), workingStore, failingCheckpointSaver{})
	result := runner.Run(
		"session-1",
		"pre_market",
		[]workflow.Step{{Name: "noop", Tool: "noop"}},
		nil,
		tools.Context{SessionID: "session-1"},
		working,
	)

	if result.Status != "failed" {
		t.Fatalf("status=%s want failed", result.Status)
	}
	if !strings.Contains(result.LastError, "checkpoint failed") {
		t.Fatalf("LastError=%q", result.LastError)
	}
}
