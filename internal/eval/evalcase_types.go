package eval

import "strings"

// EvalDialogueTurn is one scripted user turn in an eval case.
type EvalDialogueTurn struct {
	Role      string `json:"role"`
	Text      string `json:"text"`
	Judge     bool   `json:"judge,omitempty"`
	OnClarify bool   `json:"on_clarify,omitempty"`
}

// ExpectReplySpec describes how the assistant should respond on the judged turn.
type ExpectReplySpec struct {
	Rubric    string   `json:"rubric,omitempty"`
	MustCover []string `json:"must_cover,omitempty"`
	MustNot   []string `json:"must_not,omitempty"`
}

// ExpectIntentSpec is structured TurnPlan intent expectation (domain/mode/sop/act).
type ExpectIntentSpec struct {
	Domain string `json:"domain"`
	Mode   string `json:"mode"`
	SOP    bool   `json:"sop"`
	Act    string `json:"act,omitempty"`
}

// ExpectExecutionSpec references a catalog execution profile for tool checks.
type ExpectExecutionSpec struct {
	Profile     string   `json:"profile"`
	ForbidTools []string `json:"forbid_tools,omitempty"`
	// LegacyRequireTools applies judged-turn-only checks when Profile is empty.
	LegacyRequireTools []string `json:"legacy_require_tools,omitempty"`
}

// ExpectRoutingSpec is structured routing expectation (mirrors legacy flat fields).
type ExpectRoutingSpec struct {
	Domain       string   `json:"domain"`
	Mode         string   `json:"mode"`
	SOP          bool     `json:"sop"`
	RequireTools []string `json:"require_tools,omitempty"`
	ForbidTools  []string `json:"forbid_tools,omitempty"`
}

// EvalJudgeConfig controls LLM semantic judging after structured checks.
type EvalJudgeConfig struct {
	Enabled   bool    `json:"enabled"`
	ModelSlot string  `json:"model_slot,omitempty"`
	MinScore  float64 `json:"min_score,omitempty"`
}

// DialogueSnapshotTurn is one turn captured from a live session at verify time.
type DialogueSnapshotTurn struct {
	Role    string `json:"role"`
	Text    string `json:"text"`
	TurnPlan string `json:"turn_plan,omitempty"`
}

// EvalCheckResult is one verification check (routing, reply, judge, …).
type EvalCheckResult struct {
	Type     string         `json:"type"`
	Passed   bool           `json:"passed"`
	Score    *float64       `json:"score,omitempty"`
	Detail   string         `json:"detail,omitempty"`
	Expected map[string]any `json:"expected,omitempty"`
	Actual   map[string]any `json:"actual,omitempty"`
	Model    string         `json:"model,omitempty"`
}

// LiveVerifyResult is the full verify outcome for a live eval case.
type LiveVerifyResult struct {
	TurnID           string                 `json:"turn_id"`
	Passed           bool                   `json:"passed"`
	Detail           string                 `json:"detail"`
	Checks           []EvalCheckResult      `json:"checks"`
	ActualReply      string                 `json:"actual_reply,omitempty"`
	DialogueSnapshot []DialogueSnapshotTurn `json:"dialogue_snapshot,omitempty"`
	Summary          map[string]any         `json:"summary,omitempty"`
}

func buildDialogueFromLegacy(setup []string, message string) []EvalDialogueTurn {
	out := make([]EvalDialogueTurn, 0, len(setup)+1)
	for _, text := range setup {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		out = append(out, EvalDialogueTurn{Role: "user", Text: text})
	}
	if msg := strings.TrimSpace(message); msg != "" {
		out = append(out, EvalDialogueTurn{Role: "user", Text: msg, Judge: true})
	}
	return out
}

// Normalize fills dialogue / routing / judge defaults from legacy options_json fields.
func (o TurnPlanCaseOptions) Normalize() TurnPlanCaseOptions {
	out := o
	if len(out.Dialogue) == 0 {
		out.Dialogue = buildDialogueFromLegacy(out.SetupMessages, out.Message)
	} else {
		hasJudge := false
		for i := range out.Dialogue {
			if out.Dialogue[i].Judge {
				hasJudge = true
			}
		}
		if !hasJudge && len(out.Dialogue) > 0 {
			out.Dialogue[len(out.Dialogue)-1].Judge = true
		}
	}
	if out.ExpectRouting == nil && strings.TrimSpace(out.ExpectDomain) != "" {
		out.ExpectRouting = &ExpectRoutingSpec{
			Domain:       out.ExpectDomain,
			Mode:         out.ExpectMode,
			SOP:          out.ExpectSOP,
			RequireTools: append([]string(nil), out.RequireTools...),
			ForbidTools:  append([]string(nil), out.ForbidTools...),
		}
	}
	if out.ExpectIntent == nil && strings.TrimSpace(out.ExpectDomain) != "" {
		out.ExpectIntent = &ExpectIntentSpec{
			Domain: out.ExpectDomain,
			Mode:   out.ExpectMode,
			SOP:    out.ExpectSOP,
		}
	}
	if out.ExpectExecution == nil && strings.TrimSpace(out.ExecutionProfile) != "" {
		out.ExpectExecution = &ExpectExecutionSpec{
			Profile:     out.ExecutionProfile,
			ForbidTools: append([]string(nil), out.ForbidTools...),
		}
	}
	if out.ExpectExecution == nil && len(out.RequireTools) > 0 {
		out.ExpectExecution = &ExpectExecutionSpec{
			LegacyRequireTools: append([]string(nil), out.RequireTools...),
			ForbidTools:        append([]string(nil), out.ForbidTools...),
		}
	}
	if out.ExpectReply == nil {
		if spec := defaultExpectReplyForTurnID(out.TurnID); spec.Rubric != "" {
			out.ExpectReply = &spec
		}
	}
	if out.Judge == nil && out.ExpectReply != nil && strings.TrimSpace(out.ExpectReply.Rubric) != "" {
		out.Judge = &EvalJudgeConfig{Enabled: true, ModelSlot: "auxiliary", MinScore: 0.7}
	}
	if out.Judge != nil && out.Judge.MinScore <= 0 {
		out.Judge.MinScore = 0.7
	}
	return out
}

// SyncLegacyUtterances copies Dialogue back into Message / SetupMessages for runners that still read legacy fields.
func (o TurnPlanCaseOptions) SyncLegacyUtterances() TurnPlanCaseOptions {
	out := o.Normalize()
	if len(out.Dialogue) == 0 {
		return out
	}
	last := len(out.Dialogue) - 1
	out.Message = strings.TrimSpace(out.Dialogue[last].Text)
	setup := make([]string, 0, last)
	for i := 0; i < last; i++ {
		if text := strings.TrimSpace(out.Dialogue[i].Text); text != "" {
			setup = append(setup, text)
		}
	}
	out.SetupMessages = setup
	return out
}

func (o TurnPlanCaseOptions) routing() ExpectRoutingSpec {
	n := o.Normalize()
	if n.ExpectRouting != nil {
		return *n.ExpectRouting
	}
	return ExpectRoutingSpec{
		Domain: n.ExpectDomain, Mode: n.ExpectMode, SOP: n.ExpectSOP,
		RequireTools: n.RequireTools, ForbidTools: n.ForbidTools,
	}
}

func (o TurnPlanCaseOptions) intent() ExpectIntentSpec {
	n := o.Normalize()
	if n.ExpectIntent != nil {
		return *n.ExpectIntent
	}
	r := n.routing()
	return ExpectIntentSpec{Domain: r.Domain, Mode: r.Mode, SOP: r.SOP}
}

func (o TurnPlanCaseOptions) execution() ExpectExecutionSpec {
	n := o.Normalize()
	if n.ExpectExecution != nil {
		return *n.ExpectExecution
	}
	r := n.routing()
	return ExpectExecutionSpec{
		LegacyRequireTools: append([]string(nil), r.RequireTools...),
		ForbidTools:        append([]string(nil), r.ForbidTools...),
	}
}
