package tools

import (
	"strings"
	"time"
)

// PolicyAction is the permission class for a tool (Codex execpolicy aligned).
type PolicyAction string

const PolicyForbidden PolicyAction = "forbidden"

const (
	policyAllow  = PolicyAllow
	policyPrompt = PolicyPrompt
)

// ToolSpec holds optional per-tool metadata overrides. Zero values mean "infer at Register".
type ToolSpec struct {
	Policy          PolicyAction
	ReadOnly        *bool
	ConcurrencySafe *bool
	MaxResultChars  int
	Timeout         time.Duration
	DeferLoad       bool
	UserFacingLabel string
}

type resolvedMeta struct {
	Policy          PolicyAction
	ReadOnly        bool
	ConcurrencySafe bool
	MaxResultChars  int
	Timeout         time.Duration
	DeferLoad       bool
	UserFacingLabel string
}

const defaultMaxResultChars = 6000

var legacyExecutionTimeouts = map[string]time.Duration{
	"generate_dca_strategy":  4 * time.Minute,
	"generate_grid_strategy": 3 * time.Minute,
	"get_mcp_analysis":       3 * time.Minute,
	"clarify":                10 * time.Minute,
}

var perToolMaxResultChars = map[string]int{
	"get_strategy_backtest_log": 8000,
}

// resolveToolMeta merges explicit ToolSpec, YAML policy rules, and name-based defaults.
func resolveToolMeta(t Tool) resolvedMeta {
	name := t.Name
	policy := t.Spec.Policy
	if policy == "" {
		policy = inferPolicy(name)
		if override := matchPolicyRule(name); override != "" {
			policy = override
		}
	}

	readOnly := inferReadOnly(name, policy)
	if t.Spec.ReadOnly != nil {
		readOnly = *t.Spec.ReadOnly
	}

	concurrencySafe := inferConcurrencySafe(name, policy)
	if t.Spec.ConcurrencySafe != nil {
		concurrencySafe = *t.Spec.ConcurrencySafe
	}

	maxChars := defaultMaxResultChars
	if v, ok := perToolMaxResultChars[name]; ok {
		maxChars = v
	}
	if t.Spec.MaxResultChars > 0 {
		maxChars = t.Spec.MaxResultChars
	}

	timeout := legacyExecutionTimeouts[name]
	if t.Spec.Timeout > 0 {
		timeout = t.Spec.Timeout
	}

	deferLoad := t.Spec.DeferLoad
	if !deferLoad {
		deferLoad = inferDeferLoad(name)
	}

	label := t.Spec.UserFacingLabel
	if label == "" {
		label = defaultUserFacingLabel(name)
	}

	return resolvedMeta{
		Policy:          policy,
		ReadOnly:        readOnly,
		ConcurrencySafe: concurrencySafe,
		MaxResultChars:  maxChars,
		Timeout:         timeout,
		DeferLoad:       deferLoad,
		UserFacingLabel: label,
	}
}

func inferPolicy(name string) PolicyAction {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"create_", "update_", "delete_", "edit_", "switch_", "add_"} {
		if strings.HasPrefix(lower, prefix) {
			return policyPrompt
		}
	}
	return policyAllow
}

func inferReadOnly(name string, policy PolicyAction) bool {
	if policy == policyPrompt || policy == PolicyForbidden {
		return false
	}
	lower := strings.ToLower(name)
	for _, prefix := range []string{"write_", "save_"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	return true
}

func inferConcurrencySafe(name string, policy PolicyAction) bool {
	if policy == policyPrompt || policy == PolicyForbidden {
		return false
	}
	lower := strings.ToLower(name)
	switch lower {
	case "delegate_task", "delegate_tasks", "clarify", "recall", "read_working_state",
		"write_execution_log", "save_local_report", "manage_memory", "save_note",
		"update_soul", "update_preference", "create_skill":
		return false
	}
	for _, prefix := range []string{"get_", "list_", "search_", "check_", "fetch_", "probe_"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// InferConcurrencySafeForName exposes concurrency inference without a registry.
func InferConcurrencySafeForName(name string) bool {
	return inferConcurrencySafe(name, EffectivePolicyForName(name))
}

func inferDeferLoad(name string) bool {
	// Workflow-exclusive and prompt-admin tools are not in default chat toolset.
	if _, ok := reportWorkflowTools[name]; ok {
		return true
	}
	if _, ok := promptAdminTools[name]; ok {
		return true
	}
	if _, ok := analystRuntimeTools[name]; ok {
		return true
	}
	return false
}

func defaultUserFacingLabel(name string) string {
	return name
}

// EffectivePolicy returns the resolved policy for a registered tool name.
func (r *Registry) EffectivePolicy(name string) PolicyAction {
	if r == nil {
		return inferPolicy(name)
	}
	if t, ok := r.tools[name]; ok {
		return t.resolved.Policy
	}
	p := inferPolicy(name)
	if override := matchPolicyRule(name); override != "" {
		return override
	}
	return p
}

// IsReadOnly reports whether the tool is read-only after resolution.
func (r *Registry) IsReadOnly(name string) bool {
	if r != nil {
		if t, ok := r.tools[name]; ok {
			return t.resolved.ReadOnly
		}
	}
	return inferReadOnly(name, inferPolicy(name))
}

// IsConcurrencySafe reports whether parallel execution with other tools is safe.
func (r *Registry) IsConcurrencySafe(name string) bool {
	if r != nil {
		if t, ok := r.tools[name]; ok {
			return t.resolved.ConcurrencySafe
		}
	}
	return inferConcurrencySafe(name, inferPolicy(name))
}

// MaxResultChars returns the LLM-facing result byte budget for a tool.
func (r *Registry) MaxResultChars(name string) int {
	if r != nil {
		if t, ok := r.tools[name]; ok && t.resolved.MaxResultChars > 0 {
			return t.resolved.MaxResultChars
		}
	}
	if v, ok := perToolMaxResultChars[name]; ok {
		return v
	}
	return defaultMaxResultChars
}

// ResolvedMeta returns resolved metadata for inspect / dashboard.
func (r *Registry) ResolvedMeta(name string) (resolvedMeta, bool) {
	if r == nil {
		return resolvedMeta{}, false
	}
	t, ok := r.tools[name]
	if !ok {
		return resolvedMeta{}, false
	}
	return t.resolved, true
}

// SpecStats aggregates resolved metadata counts for inspect.
type SpecStats struct {
	Registered      int
	PromptTools     int
	ForbiddenTools  int
	ReadOnlyTools   int
	DeferLoadTools  int
	ConcurrencySafe int
}

// CollectSpecStats scans all registered tools.
func (r *Registry) CollectSpecStats() SpecStats {
	if r == nil {
		return SpecStats{}
	}
	var s SpecStats
	for _, name := range r.ListNames() {
		t, ok := r.tools[name]
		if !ok {
			continue
		}
		s.Registered++
		switch t.resolved.Policy {
		case policyPrompt:
			s.PromptTools++
		case PolicyForbidden:
			s.ForbiddenTools++
		}
		if t.resolved.ReadOnly {
			s.ReadOnlyTools++
		}
		if t.resolved.DeferLoad {
			s.DeferLoadTools++
		}
		if t.resolved.ConcurrencySafe {
			s.ConcurrencySafe++
		}
	}
	return s
}
