package tools_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestAnalysisToolSchemasExposeRequiredFields(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{MCP: client, WorkspaceRoot: t.TempDir()})

	byName := map[string]map[string]any{}
	for _, schema := range r.Schemas(nil) {
		byName[schema.Name] = schema.Parameters
	}

	analysis := byName["get_mcp_analysis"]
	if analysis == nil {
		t.Fatal("missing get_mcp_analysis schema")
	}
	req, _ := analysis["required"].([]any)
	if len(req) < 3 {
		t.Fatalf("get_mcp_analysis required=%v", req)
	}

	tpl := byName["get_single_prompt_template"]
	if tpl == nil {
		t.Fatal("missing get_single_prompt_template schema")
	}
	req, _ = tpl["required"].([]any)
	if len(req) == 0 {
		t.Fatalf("get_single_prompt_template required=%v", req)
	}
}

func TestGetMCPAnalysisRejectsMissingCode(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{MCP: client, WorkspaceRoot: t.TempDir()})
	res := r.Execute(tools.CallRequest{
		Name: "get_mcp_analysis", Arguments: map[string]any{"name": "SpaceX", "period": "daily"},
	}, tools.Context{SessionID: "s1", MCPToken: "mcp-test", DryRun: false})
	if res.Status != tools.StatusError {
		t.Fatalf("expected error, got %s", res.Status)
	}
}
