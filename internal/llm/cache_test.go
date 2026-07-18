package llm

import "testing"

func TestApplyCacheBreakpointsSystemAndHistory(t *testing.T) {
	t.Parallel()
	msgs := []Message{
		{Role: RoleSystem, Content: "SYSTEM"},
		{Role: RoleUser, Content: "q1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "ctx"},
		{Role: RoleUser, Content: "q2"},
	}
	out := ApplyCacheBreakpoints(msgs)
	if !out[0].CacheBreakpoint {
		t.Fatal("system should be breakpoint")
	}
	if !out[2].CacheBreakpoint {
		t.Fatalf("assistant before volatile tail should be breakpoint: %+v", out)
	}
	if out[3].CacheBreakpoint || out[4].CacheBreakpoint {
		t.Fatal("volatile tail users should not be breakpoints")
	}
}

func TestStablePrefixEndSingleUser(t *testing.T) {
	t.Parallel()
	msgs := []Message{
		{Role: RoleSystem, Content: "s"},
		{Role: RoleUser, Content: "only"},
	}
	if stablePrefixEnd(msgs) != -1 {
		t.Fatalf("want -1 for system+single user, got %d", stablePrefixEnd(msgs))
	}
}

func TestResolveExplicitPromptCacheDefault(t *testing.T) {
	t.Parallel()
	if !ResolveExplicitPromptCache(ProviderDeepSeek, nil) {
		t.Fatal("deepseek should default explicit cache on")
	}
	if ResolveExplicitPromptCache(ProviderOpenAI, nil) {
		t.Fatal("openai should default explicit cache off")
	}
	off := false
	if ResolveExplicitPromptCache(ProviderDeepSeek, &off) {
		t.Fatal("override should disable")
	}
}

func TestOpenAIProviderEmitsCacheControl(t *testing.T) {
	t.Parallel()
	p := NewOpenAIProvider(OpenAIOptions{
		Model: "deepseek-v4-flash", APIKey: "k", ExplicitPromptCache: true,
	})
	out := p.toOpenAIMessages([]Message{{Role: RoleSystem, Content: "sys", CacheBreakpoint: true}})
	content, ok := out[0]["content"].([]map[string]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content=%#v", out[0]["content"])
	}
	if content[0]["cache_control"] == nil {
		t.Fatal("expected cache_control on breakpoint")
	}
}
