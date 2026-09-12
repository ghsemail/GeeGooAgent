package eval

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

// VerifyExecution checks judged-turn and session tool traces against execution spec.
func VerifyExecution(chat *chatsession.ChatSession, spec ExpectExecutionSpec, opts TurnPlanCaseOptions) EvalCheckResult {
	trace := chatsession.TurnToolsTraceFromSession(chat)
	judged := judgedTurnTools(chat, opts, trace)
	session := chatsession.SessionToolsFromTrace(trace)
	if len(judged) == 0 {
		judged = chatsession.LastTurnToolsCalledFromSession(chat)
	}
	if len(session) == 0 {
		session = judged
	}

	check := EvalCheckResult{
		Type: "execution",
		Expected: map[string]any{
			"profile": spec.Profile,
		},
		Actual: map[string]any{
			"judged_tools":  judged,
			"session_tools": session,
		},
	}

	profileID := strings.TrimSpace(spec.Profile)
	if profileID != "" {
		check.Expected["profile"] = profileID
		ok, detail := domaincatalog.VerifyExecutionProfile(profileID, judged, session)
		check.Passed = ok
		check.Detail = detail
		if !ok {
			return check
		}
	} else {
		check.Passed = true
		check.Detail = "no execution profile"
	}

	for _, tool := range spec.ForbidTools {
		if containsString(judged, tool) {
			check.Passed = false
			check.Detail = fmt.Sprintf("forbid tool %s on judged turn", tool)
			return check
		}
	}
	for _, tool := range spec.LegacyRequireTools {
		if !containsString(judged, tool) {
			check.Passed = false
			check.Detail = fmt.Sprintf("missing tool call %s", tool)
			return check
		}
	}
	if check.Passed && check.Detail == "no execution profile" && len(spec.LegacyRequireTools) == 0 {
		check.Detail = fmt.Sprintf("judged=%s session=%s", strings.Join(judged, ","), strings.Join(session, ","))
	}
	return check
}

func verifyIntent(chat *chatsession.ChatSession, expect ExpectIntentSpec, opts TurnPlanCaseOptions) EvalCheckResult {
	r := verifyTurnPlanIntent(chat, expect, opts)
	return EvalCheckResult{
		Type:   "intent",
		Passed: r.Passed,
		Detail: r.Detail,
		Expected: map[string]any{
			"domain": expect.Domain,
			"mode":   expect.Mode,
			"sop":    expect.SOP,
			"act":    expect.Act,
		},
		Actual: map[string]any{"detail": r.Detail},
	}
}

func verifyTurnPlanIntent(chat *chatsession.ChatSession, expect ExpectIntentSpec, opts TurnPlanCaseOptions) TurnPlanResult {
	if UsesSplitClarifyScript(opts) {
		snap, ok := chatsession.FirstTurnPlanFromSession(chat)
		if !ok {
			return TurnPlanResult{Passed: false, Detail: "missing first turn plan on session"}
		}
		return matchTurnPlanSnapshot(snap, expect, opts.TurnID)
	}
	if HasJudgeTurn(opts) {
		turn := JudgedUserTurnIndex(opts, chat)
		trace := chatsession.TurnPlanTraceFromSession(chat)
		if snap, ok := chatsession.TurnPlanFromTraceAt(trace, turn); ok {
			return matchTurnPlanSnapshot(snap, expect, opts.TurnID)
		}
	}
	legacy := TurnPlanCaseOptions{
		TurnID:       opts.TurnID,
		ExpectDomain: expect.Domain,
		ExpectMode:   expect.Mode,
		ExpectSOP:    expect.SOP,
		ExpectIntent: &expect,
	}
	return VerifyTurnPlanLive(chat, legacy)
}

func judgedTurnTools(chat *chatsession.ChatSession, opts TurnPlanCaseOptions, trace []chatsession.TurnToolsEntry) []string {
	if HasJudgeTurn(opts) {
		if tools := chatsession.TurnToolsFromTraceAt(trace, JudgedUserTurnIndex(opts, chat)); len(tools) > 0 {
			return tools
		}
	}
	return chatsession.JudgedTurnToolsFromTrace(trace)
}

func matchTurnPlanSnapshot(snap chatsession.TurnPlanSnapshot, expect ExpectIntentSpec, turnID string) TurnPlanResult {
	if turnID == "" {
		turnID = "live"
	}
	res := TurnPlanResult{TurnID: turnID, Passed: true}
	var problems []string
	if snap.Domain != expect.Domain {
		problems = append(problems, fmt.Sprintf("domain=%s want %s", snap.Domain, expect.Domain))
	}
	if snap.Mode != expect.Mode {
		problems = append(problems, fmt.Sprintf("mode=%s want %s", snap.Mode, expect.Mode))
	}
	if snap.SOP != expect.SOP {
		problems = append(problems, fmt.Sprintf("sop=%v want %v", snap.SOP, expect.SOP))
	}
	if act := strings.TrimSpace(expect.Act); act != "" && strings.TrimSpace(snap.Act) != act {
		problems = append(problems, fmt.Sprintf("act=%s want %s", snap.Act, act))
	}
	if len(problems) > 0 {
		res.Passed = false
		res.Detail = strings.Join(problems, "; ")
		return res
	}
	res.Detail = fmt.Sprintf("%s/%s act=%s sop=%v", snap.Domain, snap.Mode, snap.Act, snap.SOP)
	return res
}
