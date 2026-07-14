package tools_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRecallYesterdaySummaryIsNotASuccessfulStub(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	result := r.Execute(tools.CallRequest{Name: "recall_yesterday_summary"}, tools.Context{SessionID: "test"})
	if result.Status != tools.StatusSkip {
		t.Fatalf("stub tool must not report success: status=%s summary=%s", result.Status, result.Summary)
	}
	if implemented, _ := result.Data["implemented"].(bool); implemented {
		t.Fatal("stub tool reported implemented=true")
	}
}
