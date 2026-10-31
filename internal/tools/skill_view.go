package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
)

func registerSkillViewTools(r *Registry, deps Deps) {
	if r == nil {
		return
	}
	r.Register(Tool{
		Name: "skill_view",
		Description: "Load one skill's SKILL.md (Hermes-style skill read). Does not execute workflows. " +
			"Call after picking a skill id from the injected catalog or when the user intent matches a skill description.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"skill": map[string]any{
					"type":        "string",
					"description": "Skill id (SKILL.md name), e.g. signal_diagnose, strategy-backtest-run",
				},
				"file": map[string]any{
					"type":        "string",
					"description": "Optional sibling file under the skill directory (default SKILL.md body)",
				},
			},
			"required": []any{"skill"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			if deps.SkillLoader == nil {
				return Result{Status: StatusError, Summary: "skill loader not configured"}
			}
			name := strings.TrimSpace(strArg(args, "skill", ""))
			if name == "" {
				return Result{Status: StatusError, Summary: "skill required"}
			}
			if locked := strings.TrimSpace(ctx.LockedSkill); locked != "" && !strings.EqualFold(name, locked) {
				return Result{
					Status: StatusError,
					Summary: fmt.Sprintf(
						"Monday locked_skill=%q for this turn; call skill_view with that id only (not %q) unless the user clearly changes topic",
						locked, name,
					),
				}
			}
			file := strings.TrimSpace(strArg(args, "file", ""))
			sk, body, err := deps.SkillLoader.ReadSkillFile(name, file)
			if err != nil {
				if errors.Is(err, procedural.ErrSkillNotFound) {
					return Result{Status: StatusError, Summary: "skill not found: " + name}
				}
				return Result{Status: StatusError, Summary: err.Error()}
			}
			return Result{
				Status:  StatusOK,
				Summary: "Loaded skill " + sk.Name,
				Data: map[string]any{
					"skill":       sk.Name,
					"description": sk.Description,
					"path":        sk.Path,
					"body":        body,
				},
			}
		},
	})
	r.Register(Tool{
		Name: "emit_workflow_card",
		Description: "Emit a workflow_card SSE payload for Monday / strategy UI when following a guided skill without run_chat_workflow_turn. " +
			"For signal_diagnose, pass summary fields only (stock_code, strategy_query, phase, status, diagnose_detail, probe hits); " +
			"do not embed chart_probe.bars or signal_eval — the server attaches diagnose_ui_key after storing rich payload.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"card": map[string]any{
					"type":        "object",
					"description": "workflow_card payload (skill, run_id, phase, status, stock_code, diagnose_detail, …)",
				},
			},
			"required": []any{"card"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			raw, _ := args["card"].(map[string]any)
			if len(raw) == 0 {
				return Result{Status: StatusError, Summary: "card object required"}
			}
			if deps.PrepareWorkflowCardEmit != nil {
				raw = deps.PrepareWorkflowCardEmit(ctx, raw)
			}
			if ctx.Progress != nil {
				ctx.Progress("workflow_card", raw)
			}
			skill, _ := raw["skill"].(string)
			if skill == "" {
				skill, _ = raw["template"].(string)
			}
			return Result{
				Status:  StatusOK,
				Summary: "workflow_card emitted",
				Data:    map[string]any{"skill": skill},
			}
		},
	})
}
