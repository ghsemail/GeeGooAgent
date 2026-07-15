package tools_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRecallYesterdaySummarySkipsWhenNoReport(t *testing.T) {
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	result := r.Execute(tools.CallRequest{
		Name:      "recall_yesterday_summary",
		Arguments: map[string]any{"code": "00700.HK"},
	}, tools.Context{SessionID: "test"})
	if result.Status != tools.StatusSkip {
		t.Fatalf("expected skip when no report: status=%s summary=%s", result.Status, result.Summary)
	}
	if implemented, _ := result.Data["implemented"].(bool); !implemented {
		t.Fatal("expected implemented=true")
	}
}

func TestRecallYesterdaySummaryReadsLocalReport(t *testing.T) {
	root := t.TempDir()
	reportDate := "2026-07-14"
	rel := filepath.Join("reports", reportDate, "00700.HK-premarket.md")
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Yesterday report\n\nDecision: hold"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: root})

	result := r.Execute(tools.CallRequest{
		Name: "recall_yesterday_summary",
		Arguments: map[string]any{
			"code": "00700.HK", "report_date": reportDate,
		},
	}, tools.Context{SessionID: "test", WorkspaceRoot: root})
	if result.Status != tools.StatusOK {
		t.Fatalf("status=%s summary=%s", result.Status, result.Summary)
	}
	summary, _ := result.Data["summary"].(string)
	if summary == "" || !contains(summary, "Yesterday report") {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
