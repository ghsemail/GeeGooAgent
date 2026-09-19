// Command signal_diagnose_smoke runs signal_diagnose workflow against live GeeGooSignal.
//
// Env: SIGNAL_API_URL, SIGNAL_API_KEY, SIGNAL_CATALOG_URL, SIGNAL_CATALOG_KEY (optional catalog key)
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func main() {
	sigURL := envOr("SIGNAL_API_URL", "http://146.56.225.252:3200")
	sigKey := strings.TrimSpace(os.Getenv("SIGNAL_API_KEY"))
	if sigKey == "" {
		fmt.Fprintln(os.Stderr, "SIGNAL_API_KEY required")
		os.Exit(1)
	}
	catURL := envOr("SIGNAL_CATALOG_URL", "http://146.56.225.252:3210")
	catKey := strings.TrimSpace(os.Getenv("SIGNAL_CATALOG_KEY"))

	opts := mcp.Options{AllowedHosts: []string{"146.56.225.252", "127.0.0.1", "localhost"}}
	sigClient := mcp.NewClient(sigURL, sigKey, opts)
	catClient := mcp.NewClient(catURL, catKey, opts)
	if catKey == "" {
		catClient = sigClient
	}

	reg := tools.NewRegistry()
	tools.RegisterAll(reg, tools.Deps{
		HTTP: tools.HTTPBackends{
			SignalAPI:     sigClient,
			SignalCatalog: catClient,
		},
		WeKnora:       optionalWeKnora(),
		WorkspaceRoot: os.TempDir(),
	})

	runner := &chat.Runner{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			return reg.Execute(req, toolCtx)
		},
		ComposeLLM: optionalComposeLLM(),
	}

	msg := "诊断 SAR 策略 · 腾讯"
	if len(os.Args) > 1 {
		msg = strings.Join(os.Args[1:], " ")
	}
	session := runtime.NewSession()
	toolCtx := tools.Context{MCPToken: strings.TrimSpace(os.Getenv("MCP_TOKEN"))}
	result, handled := runner.RunTurn(context.Background(), session, msg, toolCtx, 1)
	if !handled {
		fmt.Fprintln(os.Stderr, "workflow not handled")
		os.Exit(2)
	}
	if result.Failed {
		fmt.Fprintln(os.Stderr, "workflow failed:", result.Error)
		fmt.Println(result.AssistantText)
		os.Exit(1)
	}
	fmt.Println(result.AssistantText)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
