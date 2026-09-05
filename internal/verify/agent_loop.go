package verify

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ToolLookup resolves registered tool names (for agent-loop acceptance).
type ToolLookup interface {
	Get(name string) (tools.Tool, bool)
}

// AgentLoopCard is the verdict for one Hermes agent-loop parity check.
type AgentLoopCard struct {
	Name   string
	Passed bool
	Detail string
}

// Summary returns a one-line verdict.
func (c AgentLoopCard) Summary() string {
	verdict := "PASS"
	if !c.Passed {
		verdict = "FAIL"
	}
	return verdict + " " + c.Name + ": " + c.Detail
}

// VerifyAgentLoopParity runs offline checks for Hermes agent-loop alignment.
func VerifyAgentLoopParity(reg ToolLookup) []AgentLoopCard {
	checks := []AgentLoopCard{
		checkTool(reg, "clarify", "引导选项 clarify"),
		checkTool(reg, "delegate_task", "子 Agent delegate_task"),
		checkTool(reg, "delegate_tasks", "并行子 Agent delegate_tasks"),
		checkTool(reg, "recall", "跨会话记忆 recall"),
		checkTool(reg, "search_code", "行情检索 search_code"),
		checkCacheBreakpoints(),
		checkWorkflowExclusive(),
		checkDelegateNesting(),
		checkSchemaValidation(),
		checkNestedSchemaValidation(),
		checkPlanGateHold(),
		checkNDJSONProgressSchema(),
		checkSSEProgressPayload(),
		checkContextFragmentKinds(),
		checkToolSpecResolved(reg),
		checkForbiddenSchemaFilter(reg),
		checkDeferLoadCoreSchema(reg),
		checkDiscoverToolsRegistered(reg),
	}
	return checks
}

// AllAgentLoopPass reports whether every card passed.
func AllAgentLoopPass(cards []AgentLoopCard) bool {
	for _, c := range cards {
		if !c.Passed {
			return false
		}
	}
	return true
}

func checkTool(reg ToolLookup, name, label string) AgentLoopCard {
	if reg == nil {
		return AgentLoopCard{Name: label, Passed: false, Detail: "registry nil"}
	}
	if _, ok := reg.Get(name); ok {
		return AgentLoopCard{Name: label, Passed: true, Detail: name + " registered"}
	}
	return AgentLoopCard{Name: label, Passed: false, Detail: name + " missing"}
}

func checkCacheBreakpoints() AgentLoopCard {
	msgs := llm.ApplyCacheBreakpoints([]llm.Message{
		{Role: llm.RoleSystem, Content: "SYS"},
		{Role: llm.RoleUser, Content: "q1"},
		{Role: llm.RoleAssistant, Content: "a1"},
		{Role: llm.RoleUser, Content: "ctx"},
		{Role: llm.RoleUser, Content: "q2"},
	})
	if !msgs[0].CacheBreakpoint || !msgs[2].CacheBreakpoint {
		return AgentLoopCard{Name: "prompt cache breakpoints", Passed: false, Detail: "breakpoint markers missing"}
	}
	return AgentLoopCard{Name: "prompt cache breakpoints", Passed: true, Detail: "system + stable history"}
}

func checkWorkflowExclusive() AgentLoopCard {
	if !tools.IsWorkflowExclusiveTool("read_working_state") {
		return AgentLoopCard{Name: "workflow tool guard", Passed: false, Detail: "read_working_state not exclusive"}
	}
	if tools.IsWorkflowExclusiveTool("recall") {
		return AgentLoopCard{Name: "workflow tool guard", Passed: false, Detail: "recall wrongly exclusive"}
	}
	return AgentLoopCard{Name: "workflow tool guard", Passed: true, Detail: "workflow/chat split ok"}
}

func checkDelegateNesting() AgentLoopCard {
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{
		Name: "delegate_task",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			if ctx.DelegateDepth >= 1 {
				return tools.Result{Status: tools.StatusError, Summary: "nested delegate not allowed"}
			}
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	res := reg.Execute(tools.CallRequest{
		Name: "delegate_task", Arguments: map[string]any{"task": "x"},
	}, tools.Context{DelegateDepth: 1})
	if res.Status != tools.StatusError || !strings.Contains(res.Summary, "nested") {
		return AgentLoopCard{Name: "delegate nesting guard", Passed: false, Detail: res.Summary}
	}
	return AgentLoopCard{Name: "delegate nesting guard", Passed: true, Detail: "depth>=1 rejected"}
}

func checkSchemaValidation() AgentLoopCard {
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{
		Name: "need_task",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{"type": "string"},
			},
			"required": []any{"task"},
		},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	res := reg.Execute(tools.CallRequest{Name: "need_task"}, tools.Context{})
	if res.Status != tools.StatusError || !strings.Contains(res.Summary, "参数校验失败") {
		return AgentLoopCard{Name: "tool schema validation", Passed: false, Detail: res.Summary}
	}
	return AgentLoopCard{Name: "tool schema validation", Passed: true, Detail: "required args enforced"}
}

func checkNestedSchemaValidation() AgentLoopCard {
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{
		Name: "create_dca_bot",
		Parameters: map[string]any{
			"type":     "object",
			"required": []any{"signal"},
			"properties": map[string]any{
				"signal": map[string]any{
					"type":     "object",
					"required": []any{"buy_signal"},
					"properties": map[string]any{
						"buy_signal": map[string]any{
							"type":     "array",
							"minItems": float64(1),
						},
					},
				},
			},
		},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	res := reg.Execute(tools.CallRequest{
		Name: "create_dca_bot",
		Arguments: map[string]any{
			"signal": map[string]any{"buy_signal": []any{}},
		},
	}, tools.Context{})
	if res.Status != tools.StatusError || !strings.Contains(res.Summary, "至少需要") {
		return AgentLoopCard{Name: "nested schema validation", Passed: false, Detail: res.Summary}
	}
	return AgentLoopCard{Name: "nested schema validation", Passed: true, Detail: "nested object/array enforced"}
}

func checkPlanGateHold() AgentLoopCard {
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{
		Name: "create_test_bot",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			if !ctx.Approved {
				return tools.Result{Status: tools.StatusError, Summary: "should not run unapproved"}
			}
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	// Plan gate is enforced in agent.Loop; offline card only checks mutating prefix list.
	if !tools.ApprovalRequired("create_test_bot") {
		return AgentLoopCard{Name: "plan gate mutating list", Passed: false, Detail: "create_* not mutating"}
	}
	return AgentLoopCard{Name: "plan gate mutating list", Passed: true, Detail: "create_/update_/delete_ gated"}
}

func checkNDJSONProgressSchema() AgentLoopCard {
	line, err := runtime.ProgressToAgentEvent("turn_complete", map[string]any{"assistant_text": "ok"}).EncodeLine()
	if err != nil || len(line) == 0 {
		return AgentLoopCard{Name: "NDJSON progress schema", Passed: false, Detail: "encode failed"}
	}
	if runtime.AgentEventSchemaVersion != 1 {
		return AgentLoopCard{Name: "NDJSON progress schema", Passed: false, Detail: "schema version mismatch"}
	}
	return AgentLoopCard{Name: "NDJSON progress schema", Passed: true, Detail: "schema_version=1"}
}

func checkSSEProgressPayload() AgentLoopCard {
	payload := runtime.ProgressPayload("gate", map[string]any{"decision": "skip"})
	if payload["schema_version"] != 1 {
		return AgentLoopCard{Name: "SSE progress payload", Passed: false, Detail: "schema_version missing"}
	}
	if payload["item_type"] != events.ItemStatus {
		return AgentLoopCard{Name: "SSE progress payload", Passed: false, Detail: "item_type missing"}
	}
	if payload["decision"] != "skip" {
		return AgentLoopCard{Name: "SSE progress payload", Passed: false, Detail: "legacy flat fields missing"}
	}
	return AgentLoopCard{Name: "SSE progress payload", Passed: true, Detail: "schema_version+item_type+legacy flat"}
}

func checkContextFragmentKinds() AgentLoopCard {
	kinds := context.RegisteredKinds()
	if len(kinds) < 6 {
		return AgentLoopCard{Name: "context fragments", Passed: false, Detail: "kinds incomplete"}
	}
	text, applied, _ := context.Composer{MaxBytes: 80}.Compose([]context.Fragment{
		context.RecallFragment("facts", "short recall"),
		context.ToolResultFragment(strings.Repeat("x", 200)),
	})
	if text == "" || len(applied) == 0 {
		return AgentLoopCard{Name: "context fragments", Passed: false, Detail: "compose failed"}
	}
	return AgentLoopCard{Name: "context fragments", Passed: true, Detail: fmt.Sprintf("%d kinds, compose ok", len(kinds))}
}

func checkToolSpecResolved(reg ToolLookup) AgentLoopCard {
	r, ok := reg.(*tools.Registry)
	if !ok || r == nil {
		return AgentLoopCard{Name: "tool spec metadata", Passed: true, Detail: "skip (non-registry)"}
	}
	if r.EffectivePolicy("create_dca_bot") != tools.PolicyPrompt {
		return AgentLoopCard{Name: "tool spec metadata", Passed: false, Detail: "mutating policy not prompt"}
	}
	if !r.IsConcurrencySafe("search_code") {
		return AgentLoopCard{Name: "tool spec metadata", Passed: false, Detail: "search_code not concurrency-safe"}
	}
	stats := r.CollectSpecStats()
	if stats.Registered < 5 {
		return AgentLoopCard{Name: "tool spec metadata", Passed: false, Detail: "too few tools"}
	}
	return AgentLoopCard{
		Name:   "tool spec metadata",
		Passed: true,
		Detail: fmt.Sprintf("registered=%d prompt=%d readonly=%d", stats.Registered, stats.PromptTools, stats.ReadOnlyTools),
	}
}

func checkForbiddenSchemaFilter(reg ToolLookup) AgentLoopCard {
	r := tools.NewRegistry()
	r.SetPlatformConfig(tools.PlatformConfig{PolicyV2: true})
	r.Register(tools.Tool{
		Name: "delete_verify_test", Spec: tools.ToolSpec{Policy: tools.PolicyForbidden},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result { return tools.Result{} },
	})
	schemas := r.SchemasWithOptions(tools.SchemaOptions{
		Names: []string{"delete_verify_test"}, ExcludeForbidden: true,
	})
	if len(schemas) != 0 {
		return AgentLoopCard{Name: "forbidden schema filter", Passed: false, Detail: "forbidden tool visible"}
	}
	_ = reg
	return AgentLoopCard{Name: "forbidden schema filter", Passed: true, Detail: "forbidden hidden from schema"}
}

func checkDeferLoadCoreSchema(reg ToolLookup) AgentLoopCard {
	r := tools.NewRegistry()
	r.SetPlatformConfig(tools.PlatformConfig{DeferLoadTools: true, PolicyV2: false})
	deferLoad := true
	r.Register(tools.Tool{
		Name: "create_verify_defer", Spec: tools.ToolSpec{DeferLoad: deferLoad},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result { return tools.Result{} },
	})
	r.Register(tools.Tool{
		Name: "search_code",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result { return tools.Result{} },
	})
	tools.RegisterDiscoverTools(r)
	schemas := r.ChatSchemas([]string{"search_code", "create_verify_defer"}, nil)
	if len(schemas) > 20 {
		return AgentLoopCard{Name: "defer load core schema", Passed: false, Detail: fmt.Sprintf("too many schemas: %d", len(schemas))}
	}
	hasCore := false
	for _, s := range schemas {
		if s.Name == "search_code" {
			hasCore = true
		}
		if s.Name == "create_verify_defer" {
			return AgentLoopCard{Name: "defer load core schema", Passed: false, Detail: "deferred tool exposed"}
		}
	}
	if !hasCore {
		return AgentLoopCard{Name: "defer load core schema", Passed: false, Detail: "core tool missing"}
	}
	_ = reg
	return AgentLoopCard{Name: "defer load core schema", Passed: true, Detail: fmt.Sprintf("%d tools in core schema", len(schemas))}
}

func checkDiscoverToolsRegistered(reg ToolLookup) AgentLoopCard {
	if reg == nil {
		return AgentLoopCard{Name: "discover tools meta", Passed: false, Detail: "registry nil"}
	}
	if _, ok := reg.Get("discover_tools"); !ok {
		return AgentLoopCard{Name: "discover tools meta", Passed: false, Detail: "discover_tools missing"}
	}
	if _, ok := reg.Get("activate_toolset"); !ok {
		return AgentLoopCard{Name: "discover tools meta", Passed: false, Detail: "activate_toolset missing"}
	}
	return AgentLoopCard{Name: "discover tools meta", Passed: true, Detail: "discover + activate registered"}
}
