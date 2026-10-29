package agent

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/agent/decision"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SetDecisionCoordinator wires the in-loop Jev decision hook (shadow by default).
func (l *Loop) SetDecisionCoordinator(c *decision.Coordinator) {
	if l == nil {
		return
	}
	l.decision = c
}

func (l *Loop) decisionEnabled() bool {
	return l != nil && l.decision != nil && l.decision.Mode != decision.ModeOff && l.decision.Hook != nil
}

func (l *Loop) emitDecisionConsult(ctx context.Context, q decision.Question) {
	if !l.decisionEnabled() {
		return
	}
	var emit decision.ProgressFunc
	if l.onProgress != nil {
		fn := l.onProgress
		emit = func(event string, data map[string]any) { fn(event, data) }
	}
	_, _ = l.decision.Consult(ctx, q, emit)
}

func decisionQuestionID(kind decision.Kind, step int, suffix string) string {
	return fmt.Sprintf("%s:%d:%s", kind, step, suffix)
}

func (l *Loop) consultAfterLLMPlan(
	ctx context.Context,
	sessionID string,
	step int,
	resp *llm.Response,
	toolNames []string,
	mutating []llm.ToolCall,
) {
	if !l.decisionEnabled() {
		return
	}
	if len(resp.ToolCalls) == 0 {
		l.emitDecisionConsult(ctx, decision.Question{
			ID:        decisionQuestionID(decision.KindEndTurn, step, "post_llm"),
			Kind:      decision.KindEndTurn,
			Step:      step,
			SessionID: sessionID,
			Choices:   []decision.Choice{{ID: "reply", Label: "finalize assistant reply"}},
			Context: map[string]any{
				"tool_count": 0,
				"has_text":   resp.Content != "" || resp.ReasoningContent != "",
			},
		})
		return
	}

	toolChoices := make([]decision.Choice, 0, len(toolNames))
	for _, name := range toolNames {
		toolChoices = append(toolChoices, decision.Choice{ID: name, Label: name})
	}
	l.emitDecisionConsult(ctx, decision.Question{
		ID:        decisionQuestionID(decision.KindToolPlan, step, "tools"),
		Kind:      decision.KindToolPlan,
		Step:      step,
		SessionID: sessionID,
		Choices:   toolChoices,
		Context:   map[string]any{"tool_count": len(toolNames)},
	})

	if len(mutating) > 0 {
		l.emitDecisionConsult(ctx, decision.Question{
			ID:        decisionQuestionID(decision.KindPlanGate, step, "mutating"),
			Kind:      decision.KindPlanGate,
			Step:      step,
			SessionID: sessionID,
			Choices: []decision.Choice{
				{ID: "hold", Label: "hold for plan gate / user approval"},
				{ID: "proceed", Label: "proceed (requires approval path)"},
			},
			Context: map[string]any{
				"mutating_count": len(mutating),
				"mutating_tools": toolCallNames(mutating),
			},
		})
	}

	for _, call := range resp.ToolCalls {
		if call.Name != "clarify" {
			continue
		}
		choices := clarifyChoicesFromArgs(call.Arguments)
		if len(choices) == 0 {
			continue
		}
		opts := make([]decision.Choice, 0, len(choices))
		for i, ch := range choices {
			opts = append(opts, decision.Choice{ID: fmt.Sprintf("c%d", i), Label: ch})
		}
		l.emitDecisionConsult(ctx, decision.Question{
			ID:        decisionQuestionID(decision.KindClarify, step, call.ID),
			Kind:      decision.KindClarify,
			Step:      step,
			SessionID: sessionID,
			Choices:   opts,
			Context: map[string]any{
				"question": call.Arguments["question"],
			},
		})
	}
}

func clarifyChoicesFromArgs(args map[string]any) []string {
	raw, ok := args["choices"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
