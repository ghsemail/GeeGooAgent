package runtimeapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

var evalRuntimeOptionKeys = []string{
	"session_cleanup",
	"dual_model_eval",
	"random_stock_enabled",
	"stock_market",
}

func persistEvalCaseOptions(opts map[string]any) map[string]any {
	if opts == nil {
		return opts
	}
	category, _ := opts["category"].(string)
	if category != "turn_plan" {
		return opts
	}
	runtime := copyEvalRuntimeOptions(opts)
	raw, err := json.Marshal(opts)
	if err != nil {
		return opts
	}
	parsed, err := eval.ParseTurnPlanCaseOptions(raw)
	if err != nil {
		return opts
	}
	parsed = parsed.Normalize().SyncLegacyUtterances()
	out := map[string]any{}
	if err := json.Unmarshal(mustJSON(parsed), &out); err != nil {
		return opts
	}
	mergeEvalRuntimeOptions(out, runtime)
	if _, ok := out["category"]; ok {
		out["session_cleanup"] = eval.ClampSessionCleanup(fmt.Sprint(out["session_cleanup"]))
	}
	return out
}

func copyEvalRuntimeOptions(opts map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range evalRuntimeOptionKeys {
		if v, ok := opts[key]; ok && v != nil {
			out[key] = v
		}
	}
	return out
}

func mergeEvalRuntimeOptions(dst, runtime map[string]any) {
	for key, v := range runtime {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		dst[key] = v
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

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
	enrichTurnPlanGroup(row, opts)
	enrichTurnPlanUtterances(row, opts)
	enrichTurnPlanDialogue(row, opts)
}

func enrichTurnPlanGroup(row map[string]any, opts map[string]any) {
	group, _ := opts["turn_plan_group"].(string)
	group = strings.TrimSpace(group)
	if group == "" {
		return
	}
	row["turn_plan_group"] = group
	for _, cat := range eval.TurnPlanCategories() {
		if cat.ID == group {
			row["turn_plan_group_title"] = cat.Title
			return
		}
	}
	row["turn_plan_group_title"] = group
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
	if dialogue, ok := opts["dialogue"].([]any); ok && len(dialogue) > 0 {
		setup, msg := utterancesFromDialogue(dialogue)
		if msg != "" {
			row["utterance"] = msg
		}
		if len(setup) > 0 {
			row["setup_utterances"] = setup
		}
		return
	}
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

func utterancesFromDialogue(dialogue []any) (setup []string, message string) {
	for i, item := range dialogue {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(m["text"]))
		if text == "" || text == "<nil>" {
			continue
		}
		if i == len(dialogue)-1 {
			message = text
		} else {
			setup = append(setup, text)
		}
	}
	return setup, message
}
