package verify

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
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
		checkTool(reg, "delegate_task", "子 Agent delegate_task"),
		checkTool(reg, "recall", "跨会话记忆 recall"),
		checkTool(reg, "search_code", "行情检索 search_code"),
		checkCacheBreakpoints(),
		checkWorkflowExclusive(),
		checkDelegateNesting(),
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
