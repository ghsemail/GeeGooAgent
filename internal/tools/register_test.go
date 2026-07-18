package tools_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRegisterAllViaInitRegistrars(t *testing.T) {
	t.Parallel()
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})
	if len(r.Names()) != 82 {
		t.Fatalf("expected 82 builtin tools, got %d", len(r.Names()))
	}
}
