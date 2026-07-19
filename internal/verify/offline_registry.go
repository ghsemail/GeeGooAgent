package verify

import (
	"os"
	"path/filepath"

	_ "github.com/ghsemail/GeeGooAgent/internal/agent" // registers delegate_task via tools.AddRegistrar
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type offlineDelegator struct{}

func (offlineDelegator) DelegateTask(_ tools.Context, task, _ string, _ int) tools.Result {
	return tools.Result{
		Status:  tools.StatusDryRun,
		Summary: "offline verify stub delegate_task: " + task,
	}
}

func (offlineDelegator) DelegateTasks(_ tools.Context, tasks []tools.BatchDelegateTask) tools.Result {
	return tools.Result{
		Status:  tools.StatusDryRun,
		Summary: "offline verify stub delegate_tasks",
		Data:    map[string]any{"count": len(tasks)},
	}
}

// OfflineAgentLoopRegistry builds the full tool registry without config.json or network.
func OfflineAgentLoopRegistry(workspaceRoot string) *tools.Registry {
	if workspaceRoot == "" {
		workspaceRoot = filepath.Join(os.TempDir(), "geegoo-verify-agent-loop")
	}
	_ = os.MkdirAll(workspaceRoot, 0o755)
	client := mcp.NewClient("http://127.0.0.1:3120", "offline-verify", mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{
		HTTP:          tools.TestHTTPBackends(client),
		WorkspaceRoot: workspaceRoot,
		Delegate:      offlineDelegator{},
	})
	return r
}
