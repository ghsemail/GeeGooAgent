package tools

import (
	"testing"
)

func TestSchemasWithOptionsExcludeForbidden(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.SetPlatformConfig(PlatformConfig{PolicyV2: true})
	r.Register(Tool{Name: "get_foo", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	r.Register(Tool{Name: "delete_bar", Spec: ToolSpec{Policy: PolicyForbidden}, Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	schemas := r.SchemasWithOptions(SchemaOptions{
		Names:            []string{"get_foo", "delete_bar"},
		ExcludeForbidden: true,
	})
	if len(schemas) != 1 || schemas[0].Name != "get_foo" {
		t.Fatalf("schemas=%v", schemas)
	}
	schemas = r.SchemasWithOptions(SchemaOptions{
		Names: []string{"delete_bar"}, ExcludeForbidden: true,
	})
	if len(schemas) != 0 {
		t.Fatalf("single forbidden schemas=%v", schemas)
	}
}

func TestSchemasWithOptionsCoreOnly(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.SetPlatformConfig(PlatformConfig{DeferLoadTools: true})
	deferLoad := true
	r.Register(Tool{
		Name: "create_dca_bot", Spec: ToolSpec{DeferLoad: deferLoad},
		Handle: func(ctx Context, args map[string]any) Result { return Result{} },
	})
	r.Register(Tool{Name: "search_code", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	RegisterDiscoverTools(r)
	schemas := r.ChatSchemas([]string{"search_code", "create_dca_bot", "discover_tools"}, nil)
	names := map[string]bool{}
	for _, s := range schemas {
		names[s.Name] = true
	}
	if !names["search_code"] || !names["discover_tools"] {
		t.Fatalf("core missing: %v", names)
	}
	if names["create_dca_bot"] {
		t.Fatal("deferred tool should be hidden")
	}
}

func TestExecuteForbiddenPolicyV2(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.SetPlatformConfig(PlatformConfig{PolicyV2: true})
	r.Register(Tool{
		Name:   "delete_test",
		Spec:   ToolSpec{Policy: PolicyForbidden},
		Handle: func(ctx Context, args map[string]any) Result { return Result{Status: StatusOK} },
	})
	res := r.Execute(CallRequest{Name: "delete_test"}, Context{})
	if res.Status != StatusError {
		t.Fatalf("res=%+v", res)
	}
}

func TestDiscoverToolsSearch(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	RegisterDiscoverTools(r)
	r.Register(Tool{Name: "create_dca_bot", Description: "Create DCA bot", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	matches := r.SearchTools("dca", 5)
	if len(matches) == 0 {
		t.Fatal("expected matches")
	}
}

func TestActivateToolset(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	RegisterDiscoverTools(r)
	active := []string{}
	res := r.Execute(CallRequest{
		Name: "activate_toolset", Arguments: map[string]any{"toolset": "market"},
	}, Context{ActiveToolsets: &active})
	if res.Status != StatusOK {
		t.Fatalf("res=%+v", res)
	}
	if len(active) != 1 || active[0] != "market" {
		t.Fatalf("active=%v", active)
	}
}

func TestDefinitionFromTool(t *testing.T) {
	t.Parallel()
	def := DefinitionFromTool(Tool{Name: "search_code", Handle: func(ctx Context, args map[string]any) Result { return Result{} }})
	if def.Domain != DomainMarket {
		t.Fatalf("domain=%s", def.Domain)
	}
	if len(def.Toolsets) == 0 {
		t.Fatal("expected toolsets")
	}
}

func TestCheckCatalogDriftNilRegistry(t *testing.T) {
	t.Parallel()
	issues := CheckCatalogDrift(nil)
	if len(issues) == 0 {
		t.Fatal("expected issue")
	}
}

func TestRenderListSummary(t *testing.T) {
	t.Parallel()
	items := make([]any, 10)
	for i := range items {
		items[i] = map[string]any{"id": i}
	}
	out := RenderListSummary("list_foo", Result{
		Status: StatusOK, Summary: "ok", Data: map[string]any{"items": items},
	}, 4000)
	if len(out) == 0 || len(out) > 4000 {
		t.Fatalf("len=%d", len(out))
	}
}
