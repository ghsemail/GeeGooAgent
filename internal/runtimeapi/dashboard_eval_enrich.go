package runtimeapi

import (
	"encoding/json"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func enrichEvalCaseRow(row map[string]any) {
	opts, _ := row["options"].(map[string]any)
	if opts == nil {
		return
	}
	category, _ := opts["category"].(string)
	if category != "turn_plan" {
		enrichGenericEvalCaseRow(row, opts)
		return
	}
	planOnly, _ := opts["plan_only"].(bool)
	if planOnly {
		row["run_mode"] = "plan_only"
		return
	}
	row["run_mode"] = "turn_plan_live"
	enrichTurnPlanUtterances(row, opts)
	enrichTurnPlanDialogue(row, opts)
}

func enrichGenericEvalCaseRow(row map[string]any, opts map[string]any) {
	if dialogue, ok := opts["dialogue"].([]any); ok && len(dialogue) > 0 {
		row["dialogue"] = dialogue
	}
	if expectReply, ok := opts["expect_reply"].(map[string]any); ok && len(expectReply) > 0 {
		row["expect_reply"] = expectReply
	}
}

func enrichTurnPlanUtterances(row map[string]any, opts map[string]any) {
	if msg, ok := opts["message"].(string); ok && strings.TrimSpace(msg) != "" {
		row["utterance"] = strings.TrimSpace(msg)
	}
	if setup, ok := opts["setup_messages"].([]any); ok && len(setup) > 0 {
		out := make([]string, 0, len(setup))
		for _, item := range setup {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		if len(out) > 0 {
			row["setup_utterances"] = out
		}
	}
}

func enrichTurnPlanDialogue(row map[string]any, opts map[string]any) {
	if dialogue, ok := opts["dialogue"].([]any); ok && len(dialogue) > 0 {
		row["dialogue"] = dialogue
	} else {
		row["dialogue"] = legacyDialogueFromOpts(opts)
	}
	if expectReply, ok := opts["expect_reply"].(map[string]any); ok && len(expectReply) > 0 {
		row["expect_reply"] = expectReply
		return
	}
	// Synthesize display rubric from turn_id when DB row is legacy.
	raw, _ := json.Marshal(opts)
	parsed, err := eval.ParseTurnPlanCaseOptions(raw)
	if err != nil {
		return
	}
	n := parsed.Normalize()
	if n.ExpectReply != nil {
		row["expect_reply"] = n.ExpectReply
	}
	if n.ExpectRouting != nil {
		row["expect_routing"] = n.ExpectRouting
	}
	if n.Judge != nil {
		row["judge"] = n.Judge
	}
}

func legacyDialogueFromOpts(opts map[string]any) []map[string]any {
	var setup []string
	if raw, ok := opts["setup_messages"].([]any); ok {
		for _, item := range raw {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				setup = append(setup, strings.TrimSpace(s))
			}
		}
	}
	msg, _ := opts["message"].(string)
	built := eval.TurnPlanCaseOptions{SetupMessages: setup, Message: msg}.Normalize().Dialogue
	out := make([]map[string]any, 0, len(built))
	for _, turn := range built {
		out = append(out, map[string]any{
			"role": turn.Role, "text": turn.Text, "judge": turn.Judge,
		})
	}
	return out
}
