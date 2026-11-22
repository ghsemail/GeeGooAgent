package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
)

// DiagnoseSkillOptions carries POST /v1/skills/run diagnose inputs.
type DiagnoseSkillOptions struct {
	Strategy *DiagnoseStrategyInput
	Macro    *DiagnoseMacroInput
}

type DiagnoseStrategyInput struct {
	Code            string `json:"code"`
	SignalFrequency string `json:"signal_frequency"`
	AsOf            string `json:"as_of,omitempty"`
}

type DiagnoseMacroInput struct {
	Code string `json:"code"`
	AsOf string `json:"as_of,omitempty"`
}

// DiagnoseSkillResult is returned from diagnose_strategy / diagnose_macro skills.
type DiagnoseSkillResult struct {
	Status    string                `json:"status"`
	SessionID string                `json:"session_id,omitempty"`
	Skill     string                `json:"skill"`
	Result    diagnoseskill.Result  `json:"result,omitempty"`
	Error     string                `json:"error,omitempty"`
}

// RunDiagnoseSkill executes diagnose_strategy or diagnose_macro (not Agent Loop).
func (a *App) RunDiagnoseSkill(ctx context.Context, skill string, opts DiagnoseSkillOptions) (DiagnoseSkillResult, error) {
	if a == nil || a.Config == nil {
		return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "app not ready"}, fmt.Errorf("app not ready")
	}
	sessionID := newSessionID()
	runID := sessionID
	lanes := &diagnoseskill.LaneClient{
		BaseURL: a.Config.MacroAPIURL(),
		APIKey:  a.Config.MacroAPIKey(),
	}
	jev := a.NewStrategyDevJevClient()

	var in diagnoseskill.RunInput
	in.Lanes = lanes
	in.JEV = jev
	in.SessionID = sessionID
	in.RunID = runID

	switch skill {
	case "diagnose_strategy":
		if opts.Strategy == nil || strings.TrimSpace(opts.Strategy.Code) == "" {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "strategy.code is required"},
				fmt.Errorf("strategy.code is required")
		}
		if strings.TrimSpace(opts.Strategy.SignalFrequency) == "" {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "strategy.signal_frequency is required"},
				fmt.Errorf("strategy.signal_frequency is required")
		}
		in.Code = strings.TrimSpace(opts.Strategy.Code)
		in.SignalFreq = strings.TrimSpace(opts.Strategy.SignalFrequency)
		in.AsOf = strings.TrimSpace(opts.Strategy.AsOf)
		in.Macro = false
	case "diagnose_macro":
		if opts.Macro == nil || strings.TrimSpace(opts.Macro.Code) == "" {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "macro.code is required"},
				fmt.Errorf("macro.code is required")
		}
		in.Code = strings.TrimSpace(opts.Macro.Code)
		in.AsOf = strings.TrimSpace(opts.Macro.AsOf)
		in.Macro = true
	default:
		return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "unknown diagnose skill"},
			fmt.Errorf("unknown diagnose skill: %s", skill)
	}

	a.EventBus.Emit("RunStarted", map[string]any{"session_id": sessionID, "skill": skill, "code": in.Code})

	result, err := diagnoseskill.Run(ctx, in)
	if err != nil {
		return DiagnoseSkillResult{
			Status: "failed", SessionID: sessionID, Skill: skill, Error: err.Error(),
		}, err
	}
	return DiagnoseSkillResult{
		Status: "ok", SessionID: sessionID, Skill: skill, Result: result,
	}, nil
}