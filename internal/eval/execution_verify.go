package eval

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

// VerifyExecution checks judged-turn and session tool traces against execution spec.
func VerifyExecution(chat *chatsession.ChatSession, spec ExpectExecutionSpec) EvalCheckResult {
	trace := chatsession.TurnToolsTraceFromSession(chat)
	judged := chatsession.JudgedTurnToolsFromTrace(trace)
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

func verifyIntent(chat *chatsession.ChatSession, expect ExpectIntentSpec) EvalCheckResult {
	legacy := TurnPlanCaseOptions{
		ExpectDomain: expect.Domain,
		ExpectMode:   expect.Mode,
		ExpectSOP:    expect.SOP,
		ExpectIntent: &expect,
	}
	r := VerifyTurnPlanLive(chat, legacy)
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
