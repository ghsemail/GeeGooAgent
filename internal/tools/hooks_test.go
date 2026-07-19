package tools_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestHookRunnerFailClosed(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("hook scripts are POSIX sh on CI")
	}
	script := filepath.Join(t.TempDir(), "fail.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &tools.HookRunner{ToolBefore: []string{script}, FailClosed: true}
	r := tools.NewRegistry()
	r.Register(tools.Tool{
		Name: "noop",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	ctx := tools.Context{Hooks: runner}
	res := r.Execute(tools.CallRequest{Name: "noop"}, ctx)
	if res.Status != tools.StatusError || !strings.Contains(res.Summary, "hook failed") {
		t.Fatalf("result=%+v", res)
	}
}
