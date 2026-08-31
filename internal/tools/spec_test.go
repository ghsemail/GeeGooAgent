package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveToolMetaMutating(t *testing.T) {
	t.Parallel()
	meta := resolveToolMeta(Tool{Name: "create_dca_bot"})
	if meta.Policy != PolicyPrompt {
		t.Fatalf("policy=%s", meta.Policy)
	}
	if meta.ReadOnly {
		t.Fatal("mutating tool should not be read-only")
	}
	if meta.ConcurrencySafe {
		t.Fatal("mutating tool should not be concurrency-safe")
	}
}

func TestResolveToolMetaReadOnly(t *testing.T) {
	t.Parallel()
	meta := resolveToolMeta(Tool{Name: "search_code"})
	if meta.Policy != PolicyAllow {
		t.Fatalf("policy=%s", meta.Policy)
	}
	if !meta.ReadOnly {
		t.Fatal("search_code should be read-only")
	}
	if !meta.ConcurrencySafe {
		t.Fatal("search_code should be concurrency-safe")
	}
}

func TestResolveToolMetaTimeoutOverride(t *testing.T) {
	t.Parallel()
	meta := resolveToolMeta(Tool{Name: "clarify"})
	if meta.Timeout != 10*time.Minute {
		t.Fatalf("timeout=%v", meta.Timeout)
	}
}

func TestRegistryCollectSpecStats(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(Tool{Name: "get_foo", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	r.Register(Tool{Name: "create_bar", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	stats := r.CollectSpecStats()
	if stats.Registered != 2 || stats.PromptTools != 1 || stats.ReadOnlyTools < 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestLoadPolicyFileJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "tool_policy.json")
	if err := os.WriteFile(path, []byte(`{"rules":[{"match":"delete_*","action":"forbidden"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadPolicyFile(path); err != nil {
		t.Fatal(err)
	}
	if got := matchPolicyRule("delete_dca_bot"); got != PolicyForbidden {
		t.Fatalf("got %s", got)
	}
}

func TestMatchGlobAlternation(t *testing.T) {
	t.Parallel()
	if !matchGlob("list_foo", "get_*|list_*") {
		t.Fatal("expected match")
	}
}

func TestRenderResultForLLMTruncates(t *testing.T) {
	t.Parallel()
	big := make([]byte, 8000)
	for i := range big {
		big[i] = 'x'
	}
	out := RenderResultForLLM("list_foo", Result{Status: StatusOK, Summary: string(big)}, 1000)
	if len(out) > 1000 {
		t.Fatalf("len=%d", len(out))
	}
}
