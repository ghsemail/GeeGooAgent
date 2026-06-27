package tools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestListSmartTradesSendsMCPToken(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getAllSmartTrades" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":100,"data":[],"message":"success"}`))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{MCP: client, WorkspaceRoot: t.TempDir()})

	result := r.Execute(tools.CallRequest{Name: "list_smart_trades", Arguments: map[string]any{}}, tools.Context{
		SessionID: "s1", MCPToken: "mcp-test-user", DryRun: false,
	})
	if result.Status != tools.StatusOK {
		t.Fatalf("status=%s summary=%s", result.Status, result.Summary)
	}
	if gotBody["mcp_token"] != "mcp-test-user" {
		t.Fatalf("body mcp_token = %v", gotBody["mcp_token"])
	}
}
