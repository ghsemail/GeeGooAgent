package tools_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

func TestCatalogHTTPCount(t *testing.T) {
	specs := catalog.AllHTTP()
	if len(specs) != 61 {
		t.Fatalf("expected 61 HTTP specs, got %d", len(specs))
	}
}

func TestRegisterAllToolCount(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})
	names := r.Names()
	if len(names) != 83 {
		t.Fatalf("expected 83 tools, got %d", len(names))
	}
}

func TestAllToolsDryRun(t *testing.T) {
	root := t.TempDir()
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: root})
	state := infra.NewStateStore(filepath.Join(root, "state"))
	ctx := tools.Context{
		SessionID: "test", MCPToken: "tok", DryRun: true, WorkspaceRoot: root, StateStore: state,
	}
	for _, name := range r.Names() {
		if name == "read_working_state" || name == "clarify" || name == "delegate_task" || name == "delegate_tasks" {
			continue
		}
		result := r.Execute(tools.CallRequest{Name: name, Arguments: map[string]any{
			"code": "00700.HK", "name": "腾讯控股", "stock_name": "腾讯控股", "regex": "00700", "query": "SpaceX IPO", "period": "daily",
			"task": "dry-run probe", "botname": "dry-run-bot",
		}}, ctx)
		if result.Status == tools.StatusError {
			if strings.Contains(result.Summary, "参数校验失败") {
				continue
			}
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

func TestNewsToolsSkipWhenScriptRunnerUnavailable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("GEEGOO_NEWS_DISABLE_GO", "1")
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: root, ProjectRoot: root})

	ctx := tools.Context{SessionID: "test", MCPToken: "tok", WorkspaceRoot: root}
	cases := []tools.CallRequest{
		{Name: "fetch_market_news", Arguments: map[string]any{"market": "US"}},
		{Name: "fetch_stock_news", Arguments: map[string]any{"code": "00700.HK"}},
	}
	for _, tc := range cases {
		result := r.Execute(tc, ctx)
		if result.Status != tools.StatusSkip {
			t.Fatalf("%s status=%s summary=%s", tc.Name, result.Status, result.Summary)
		}
	}
}

func TestLoopbackStrategyUsesDirectResponse(t *testing.T) {
	t.Parallel()
	for _, spec := range catalog.AllHTTP() {
		if spec.Name != "loopback_strategy" {
			continue
		}
		if !spec.DirectResponse {
			t.Fatal("loopback_strategy must use DirectResponse (signal-api returns bare JSON, not MCP envelope)")
		}
		return
	}
	t.Fatal("loopback_strategy not found in HTTP catalog")
}
