package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultHookTimeout = 5 * time.Second

// HookRunner executes configured shell hooks around tool calls.
type HookRunner struct {
	ToolBefore []string
	ToolAfter  []string
	FailClosed bool
	Timeout    time.Duration
}

// HookPayload is passed to hook scripts on stdin.
type HookPayload struct {
	Phase     string         `json:"phase"`
	Tool      string         `json:"tool"`
	SessionID string         `json:"session_id,omitempty"`
	Step      int            `json:"step,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Status    string         `json:"status,omitempty"`
	Summary   string         `json:"summary,omitempty"`
}

// RunToolBefore invokes tool_before hooks. Returns error when FailClosed and hook fails.
func (h *HookRunner) RunToolBefore(ctx Context, toolName string, args map[string]any) error {
	if h == nil || len(h.ToolBefore) == 0 {
		return nil
	}
	payload := HookPayload{
		Phase: "tool_before", Tool: toolName, SessionID: ctx.SessionID,
		Step: ctx.Step, Arguments: args,
	}
	return h.runAll(ctx.GoContext(), h.ToolBefore, payload)
}

// RunToolAfter invokes tool_after hooks.
func (h *HookRunner) RunToolAfter(ctx Context, toolName string, args map[string]any, result Result) error {
	if h == nil || len(h.ToolAfter) == 0 {
		return nil
	}
	payload := HookPayload{
		Phase: "tool_after", Tool: toolName, SessionID: ctx.SessionID,
		Step: ctx.Step, Arguments: args, Status: string(result.Status), Summary: result.Summary,
	}
	return h.runAll(ctx.GoContext(), h.ToolAfter, payload)
}

func (h *HookRunner) runAll(ctx context.Context, scripts []string, payload HookPayload) error {
	if h == nil {
		return nil
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = defaultHookTimeout
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var errs []string
	for _, script := range scripts {
		script = strings.TrimSpace(script)
		if script == "" {
			continue
		}
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		cmd := exec.CommandContext(runCtx, script)
		cmd.Stdin = bytes.NewReader(raw)
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", script, err))
		}
		cancel()
	}
	if len(errs) == 0 {
		return nil
	}
	err = fmt.Errorf("hook failed: %s", strings.Join(errs, "; "))
	if h.FailClosed {
		return err
	}
	return nil
}
