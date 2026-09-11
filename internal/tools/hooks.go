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
const maxHookInjectBytes = 2048

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

// RunToolBefore invokes tool_before hooks. Returns inject text and error when FailClosed and hook fails.
func (h *HookRunner) RunToolBefore(ctx Context, toolName string, args map[string]any) (string, error) {
	if h == nil || len(h.ToolBefore) == 0 {
		return "", nil
	}
	payload := HookPayload{
		Phase: "tool_before", Tool: toolName, SessionID: ctx.SessionID,
		Step: ctx.Step, Arguments: args,
	}
	return h.runAll(ctx.GoContext(), h.ToolBefore, payload)
}

// RunToolAfter invokes tool_after hooks.
func (h *HookRunner) RunToolAfter(ctx Context, toolName string, args map[string]any, result Result) (string, error) {
	if h == nil || len(h.ToolAfter) == 0 {
		return "", nil
	}
	payload := HookPayload{
		Phase: "tool_after", Tool: toolName, SessionID: ctx.SessionID,
		Step: ctx.Step, Arguments: args, Status: string(result.Status), Summary: result.Summary,
	}
	return h.runAll(ctx.GoContext(), h.ToolAfter, payload)
}

func (h *HookRunner) runAll(ctx context.Context, scripts []string, payload HookPayload) (string, error) {
	if h == nil {
		return "", nil
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = defaultHookTimeout
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	var injectParts []string
	var errs []string
	for _, script := range scripts {
		script = strings.TrimSpace(script)
		if script == "" {
			continue
		}
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		var stdout bytes.Buffer
		cmd := exec.CommandContext(runCtx, script)
		cmd.Stdin = bytes.NewReader(raw)
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", script, err))
		} else if text := parseHookInject(stdout.String()); text != "" {
			injectParts = append(injectParts, text)
		}
		cancel()
	}
	inject := strings.Join(injectParts, "\n")
	if len(inject) > maxHookInjectBytes {
		inject = inject[:maxHookInjectBytes]
	}
	if len(errs) == 0 {
		return inject, nil
	}
	err = fmt.Errorf("hook failed: %s", strings.Join(errs, "; "))
	if h.FailClosed {
		return inject, err
	}
	return inject, nil
}

func parseHookInject(stdout string) string {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return ""
	}
	var payload struct {
		Inject string `json:"inject"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err == nil && strings.TrimSpace(payload.Inject) != "" {
		return strings.TrimSpace(payload.Inject)
	}
	return stdout
}
