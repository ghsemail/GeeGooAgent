package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
)

// DiagnoseSkillOptions carries POST /v1/skills/run diagnose inputs.
type DiagnoseSkillOptions struct {
	Strategy      *DiagnoseStrategyInput
	Macro         *DiagnoseMacroInput
	MacroPool     *DiagnoseMacroPoolInput
	MacroOverall  *DiagnoseMacroOverallInput
	MacroNews     *diagnoseskill.MacroNewsRequest
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

// DiagnoseMacroPoolSymbol is one symbol line for pool JEV skill.
type DiagnoseMacroPoolSymbol struct {
	Symbol       string  `json:"symbol"`
	ResultID     string  `json:"result_id"`
	JEVDirection string  `json:"jev_direction"`
	JEVReason    string  `json:"jev_reason"`
	Confidence   float64 `json:"confidence,omitempty"`
}

// DiagnoseMacroOverallPool is one pool row for overall JEV skill.
type DiagnoseMacroOverallPool struct {
	PoolWorkflow string `json:"pool_workflow"`
	JEVDirection string `json:"jev_direction"`
	JEVReason    string `json:"jev_reason"`
}

// DiagnoseMacroOverallInput is workflow-assembled overall tier payload.
type DiagnoseMacroOverallInput struct {
	Date  string                     `json:"date"`
	Pools []DiagnoseMacroOverallPool `json:"pools"`
}

// DiagnoseMacroPoolInput is workflow-assembled pool tier payload.
type DiagnoseMacroPoolInput struct {
	Date             string                    `json:"date"`
	PoolWorkflow     string                    `json:"pool_workflow"`
	SourceBatchID    string                    `json:"source_batch_id"`
	Title            string                    `json:"title,omitempty"`
	ExpectedSymbols  []string                  `json:"expected_symbols"`
	Symbols          []DiagnoseMacroPoolSymbol `json:"symbols"`
}

// DiagnoseSkillResult is returned from diagnose_strategy / diagnose_macro skills.
type DiagnoseSkillResult struct {
	Status    string                `json:"status"`
	SessionID string                `json:"session_id,omitempty"`
	Skill     string                `json:"skill"`
	Result    diagnoseskill.Result  `json:"result,omitempty"`
	Error     string                `json:"error,omitempty"`
}

func trimStringList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
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
	case diagnoseskill.SkillMacroNews:
		if opts.MacroNews == nil {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "macro_news body is required"},
				fmt.Errorf("macro_news body is required")
		}
		a.EventBus.Emit("RunStarted", map[string]any{
			"session_id": sessionID, "skill": skill, "pool_id": opts.MacroNews.PoolID,
		})
		result, err := diagnoseskill.RunMacroNews(ctx, diagnoseskill.MacroNewsRunInput{
			Request: *opts.MacroNews, LLM: a.MacroNewsLLM(), JEV: jev,
		})
		if err != nil {
			return DiagnoseSkillResult{
				Status: "failed", SessionID: sessionID, Skill: skill, Error: err.Error(),
			}, err
		}
		return DiagnoseSkillResult{
			Status: "ok", SessionID: sessionID, Skill: skill, Result: result,
		}, nil
	case "diagnose_macro_overall":
		if opts.MacroOverall == nil {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "macro_overall body is required"},
				fmt.Errorf("macro_overall body is required")
		}
		overallIn := diagnoseskill.MacroOverallRunInput{
			Date:         strings.TrimSpace(opts.MacroOverall.Date),
			SessionID:    sessionID,
			RunID:        runID,
			JEV:          jev,
			MinReasonLen: 20,
		}
		for _, p := range opts.MacroOverall.Pools {
			overallIn.Pools = append(overallIn.Pools, diagnoseskill.MacroOverallPoolRow{
				PoolWorkflow: strings.TrimSpace(p.PoolWorkflow),
				JEVDirection: strings.TrimSpace(p.JEVDirection),
				JEVReason:    strings.TrimSpace(p.JEVReason),
			})
		}
		a.EventBus.Emit("RunStarted", map[string]any{
			"session_id": sessionID, "skill": skill, "date": overallIn.Date,
		})
		result, err := diagnoseskill.RunOverall(ctx, overallIn)
		if err != nil {
			return DiagnoseSkillResult{
				Status: "failed", SessionID: sessionID, Skill: skill, Error: err.Error(),
			}, err
		}
		return DiagnoseSkillResult{
			Status: "ok", SessionID: sessionID, Skill: skill, Result: result,
		}, nil
	case "diagnose_macro_pool":
		if opts.MacroPool == nil {
			return DiagnoseSkillResult{Status: "failed", Skill: skill, Error: "macro_pool body is required"},
				fmt.Errorf("macro_pool body is required")
		}
		poolIn := diagnoseskill.MacroPoolRunInput{
			Date:             strings.TrimSpace(opts.MacroPool.Date),
			PoolWorkflow:     strings.TrimSpace(opts.MacroPool.PoolWorkflow),
			SourceBatchID:    strings.TrimSpace(opts.MacroPool.SourceBatchID),
			Title:            strings.TrimSpace(opts.MacroPool.Title),
			ExpectedSymbols: trimStringList(opts.MacroPool.ExpectedSymbols),
			SessionID:        sessionID,
			RunID:            runID,
			JEV:              jev,
			MinReasonLen:     20,
		}
		for _, s := range opts.MacroPool.Symbols {
			poolIn.Symbols = append(poolIn.Symbols, diagnoseskill.MacroPoolSymbolRow{
				Symbol:       strings.TrimSpace(s.Symbol),
				ResultID:     strings.TrimSpace(s.ResultID),
				JEVDirection: strings.TrimSpace(s.JEVDirection),
				JEVReason:    strings.TrimSpace(s.JEVReason),
				Confidence:   s.Confidence,
			})
		}
		a.EventBus.Emit("RunStarted", map[string]any{
			"session_id": sessionID, "skill": skill, "pool_workflow": poolIn.PoolWorkflow,
		})
		result, err := diagnoseskill.RunPool(ctx, poolIn)
		if err != nil {
			return DiagnoseSkillResult{
				Status: "failed", SessionID: sessionID, Skill: skill, Error: err.Error(),
			}, err
		}
		return DiagnoseSkillResult{
			Status: "ok", SessionID: sessionID, Skill: skill, Result: result,
		}, nil
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