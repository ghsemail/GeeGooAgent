package tools_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

func TestCatalogHTTPCount(t *testing.T) {
	specs := catalog.AllHTTP()
	if len(specs) < 60 {
		t.Fatalf("expected >= 60 HTTP specs, got %d", len(specs))
	}
}

func TestRegisterAllToolCount(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{MCP: client, WorkspaceRoot: t.TempDir()})
	names := r.Names()
	if len(names) < 80 {
		t.Fatalf("expected >= 80 tools, got %d", len(names))
	}
}

func TestAllToolsDryRun(t *testing.T) {
	root := t.TempDir()
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{MCP: client, WorkspaceRoot: root})
	state := infra.NewStateStore(filepath.Join(root, "state"))
	ctx := tools.Context{
		SessionID: "test", MCPToken: "tok", DryRun: true, WorkspaceRoot: root, StateStore: state,
	}
	for _, name := range r.Names() {
		if name == "read_working_state" {
			continue
		}
		result := r.Execute(tools.CallRequest{Name: name, Arguments: map[string]any{
			"code": "00700.HK", "regex": "00700", "query": "SpaceX IPO",
		}}, ctx)
		if result.Status == tools.StatusError {
			t.Fatalf("%s dry-run failed: %s", name, result.Summary)
		}
		if name == "fetch_market_news" || name == "fetch_stock_news" {
			if result.Status != tools.StatusDryRun {
				t.Fatalf("%s expected dry_run in test", name)
			}
			continue
		}
	}
}
