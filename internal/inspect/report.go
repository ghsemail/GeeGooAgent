package inspect

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/verify"
)

// Options configures inspect output.
type Options struct {
	ConfigPath string
	QuickLoop  bool // run verify agent-loop cards
}

// Report is a read-only snapshot of agent loop configuration.
type Report struct {
	ConfigPath string
	Profile    string
	LLM        LLMSection
	Loop       LoopSection
	Toolsets   []ToolsetSection
	Tools      ToolsSection
	Skills     []string
	Runtime    RuntimeSection
	Verify     []string
}

type LLMSection struct {
	Provider string
	Model    string
	Thinking bool
	DryRun   bool
}

type LoopSection struct {
	MaxSteps            int
	SubAgentMaxSteps    int
	ToolMaxParallel     int
	DelegateMaxParallel int
	ToolTimeoutSec      int
	CompressEnabled     bool
	CompressThreshold   float64
	HygieneThreshold    float64
	ContextLength       int
	PlanGate            bool
	HooksConfigured     bool
	HooksBefore         int
	HooksAfter          int
	HooksFailClosed     bool
}

type ToolsetSection struct {
	ID          string
	Label       string
	Enabled     bool
	ToolCount   int
}

type ToolsSection struct {
	Registered int
	ChatActive int
	WorkflowExclusive int
}

type RuntimeSection struct {
	GatewayConfigured bool
	MCPBase           string
	OutputDir         string
}

// Build collects inspect data from a loaded application.
func Build(application *app.App, opts Options) Report {
	r := Report{ConfigPath: opts.ConfigPath}
	if application == nil {
		return r
	}
	cfg := application.Config
	if cfg != nil {
		r.Profile = cfg.ResolvedProfile
		thinking := false
		if cfg.LLM.Thinking != nil {
			thinking = *cfg.LLM.Thinking
		}
		r.LLM = LLMSection{
			Provider: cfg.LLM.Provider,
			Model:    cfg.LLM.Model,
			Thinking: thinking,
			DryRun:   cfg.DryRun,
		}
		comp := cfg.EffectiveCompression()
		r.Loop = LoopSection{
			MaxSteps:            cfg.EffectiveMaxSteps(),
			SubAgentMaxSteps:    cfg.EffectiveSubAgentMaxSteps(),
			ToolMaxParallel:     cfg.EffectiveToolMaxParallel(),
			DelegateMaxParallel: cfg.EffectiveDelegateMaxParallel(),
			ToolTimeoutSec:      int(cfg.EffectiveToolTimeout().Seconds()),
			CompressEnabled:     comp.Enabled,
			CompressThreshold:   comp.Threshold,
			HygieneThreshold:    comp.HygieneThreshold,
			ContextLength:       comp.ContextLength,
			PlanGate:            cfg.EffectivePlanGate(),
			HooksConfigured:     application.Hooks != nil,
			HooksBefore:         len(cfg.Hooks.ToolBefore),
			HooksAfter:          len(cfg.Hooks.ToolAfter),
			HooksFailClosed:     cfg.Hooks.FailClosed,
		}
		r.Runtime.OutputDir = cfg.OutputDir
	}
	if application.Gateway != nil {
		r.Runtime.GatewayConfigured = true
	}
	if application.MCP != nil && cfg != nil {
		r.Runtime.MCPBase = maskURL(cfg.EffectiveMCPURL())
	}
	if application.Registry != nil {
		r.Tools.Registered = len(application.Registry.ListNames())
		chatNames := application.ChatToolNames()
		r.Tools.ChatActive = len(chatNames)
		r.Tools.WorkflowExclusive = len(tools.WorkflowExclusiveToolNames())
	}
	enabled := map[string]bool{}
	if cfg != nil {
		for _, id := range cfg.EffectiveChatToolsets() {
			enabled[strings.ToLower(id)] = true
		}
	}
	for _, ts := range tools.AllToolsets() {
		r.Toolsets = append(r.Toolsets, ToolsetSection{
			ID: ts.ID, Label: ts.Label,
			Enabled: enabled[ts.ID] || (len(enabled) == 0 && ts.ChatDefault),
			ToolCount: len(ts.Names()),
		})
	}
	for _, sk := range app.DefaultSkills.List() {
		r.Skills = append(r.Skills, sk.Name)
	}
	if opts.QuickLoop && application.Registry != nil {
		for _, card := range verify.VerifyAgentLoopParity(application.Registry) {
			r.Verify = append(r.Verify, card.Summary())
		}
	}
	return r
}

// FormatText renders a human-readable report.
func FormatText(r Report) string {
	var b strings.Builder
	b.WriteString("GeeGoo Agent Inspect\n")
	if r.ConfigPath != "" {
		b.WriteString(fmt.Sprintf("config: %s\n", r.ConfigPath))
	}
	if r.Profile != "" {
		b.WriteString(fmt.Sprintf("profile: %s\n", r.Profile))
	}
	b.WriteByte('\n')
	b.WriteString("[LLM]\n")
	b.WriteString(fmt.Sprintf("  provider: %s\n", r.LLM.Provider))
	b.WriteString(fmt.Sprintf("  model: %s\n", r.LLM.Model))
	b.WriteString(fmt.Sprintf("  thinking: %v  dry_run: %v\n", r.LLM.Thinking, r.LLM.DryRun))
	b.WriteByte('\n')
	b.WriteString("[Agent Loop]\n")
	b.WriteString(fmt.Sprintf("  max_steps: %d  sub_agent_max_steps: %d\n", r.Loop.MaxSteps, r.Loop.SubAgentMaxSteps))
	b.WriteString(fmt.Sprintf("  tool_max_parallel: %d  delegate_max_parallel: %d  tool_timeout_sec: %d\n",
		r.Loop.ToolMaxParallel, r.Loop.DelegateMaxParallel, r.Loop.ToolTimeoutSec))
	b.WriteString(fmt.Sprintf("  plan_gate: %v  hooks: configured=%v before=%d after=%d fail_closed=%v\n",
		r.Loop.PlanGate, r.Loop.HooksConfigured, r.Loop.HooksBefore, r.Loop.HooksAfter, r.Loop.HooksFailClosed))
	b.WriteString(fmt.Sprintf("  compression: enabled=%v threshold=%.2f hygiene=%.2f context_length=%d\n",
		r.Loop.CompressEnabled, r.Loop.CompressThreshold, r.Loop.HygieneThreshold, r.Loop.ContextLength))
	b.WriteByte('\n')
	b.WriteString("[Tools]\n")
	b.WriteString(fmt.Sprintf("  registered: %d  chat_active: %d  workflow_exclusive: %d\n",
		r.Tools.Registered, r.Tools.ChatActive, r.Tools.WorkflowExclusive))
	b.WriteByte('\n')
	b.WriteString("[Toolsets]\n")
	for _, ts := range r.Toolsets {
		flag := " "
		if ts.Enabled {
			flag = "*"
		}
		b.WriteString(fmt.Sprintf("  %s %-18s (%d tools) %s\n", flag, ts.ID, ts.ToolCount, ts.Label))
	}
	b.WriteByte('\n')
	b.WriteString("[Skills]\n")
	if len(r.Skills) == 0 {
		b.WriteString("  (none)\n")
	} else {
		b.WriteString("  " + strings.Join(r.Skills, ", ") + "\n")
	}
	b.WriteByte('\n')
	b.WriteString("[Runtime]\n")
	b.WriteString(fmt.Sprintf("  gateway: %v  mcp: %s\n", r.Runtime.GatewayConfigured, r.Runtime.MCPBase))
	if r.Runtime.OutputDir != "" {
		b.WriteString(fmt.Sprintf("  output_dir: %s\n", r.Runtime.OutputDir))
	}
	if len(r.Verify) > 0 {
		b.WriteByte('\n')
		b.WriteString("[verify agent-loop]\n")
		for _, line := range r.Verify {
			b.WriteString("  " + line + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func maskURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "(unset)"
	}
	if i := strings.Index(raw, "://"); i >= 0 {
		return raw[:i+3] + "***"
	}
	return "***"
}
