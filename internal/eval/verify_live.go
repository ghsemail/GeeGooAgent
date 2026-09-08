package eval

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// VerifyTurnPlanLiveFull runs routing/tools/reply/judge checks for a completed session.
func VerifyTurnPlanLiveFull(ctx context.Context, chat *chatsession.ChatSession, opts TurnPlanCaseOptions, judge ReplyJudge) LiveVerifyResult {
	opts = opts.Normalize()
	turnID := opts.TurnID
	if turnID == "" {
		turnID = "live"
	}
	res := LiveVerifyResult{
		TurnID:           turnID,
		Passed:           true,
		DialogueSnapshot: DialogueSnapshotFromSession(chat),
		Checks:           []EvalCheckResult{},
		Summary:          map[string]any{},
	}
	if chat == nil {
		res.Passed = false
		res.Detail = "session not found"
		return res
	}

	intent := verifyIntent(chat, opts.intent())
	res.Checks = append(res.Checks, intent)

	execution := VerifyExecution(chat, opts.execution())
	if strings.TrimSpace(opts.execution().Profile) != "" || len(opts.execution().LegacyRequireTools) > 0 || len(opts.execution().ForbidTools) > 0 {
		res.Checks = append(res.Checks, execution)
	}

	actualReply := LastAssistantReply(chat)
	res.ActualReply = actualReply

	if lengthCheck := verifyReplyLengthCheck(actualReply, opts.MinReplyChars); lengthCheck != nil {
		res.Checks = append(res.Checks, *lengthCheck)
	}
	if kwCheck := verifyPassKeywordsCheck(actualReply, opts.PassKeywords); kwCheck != nil {
		res.Checks = append(res.Checks, *kwCheck)
	}
	if opts.ExpectReply != nil {
		if coverCheck := verifyMustCoverCheck(actualReply, opts.ExpectReply); coverCheck != nil {
			res.Checks = append(res.Checks, *coverCheck)
		}
		if notCheck := verifyMustNotCheck(actualReply, opts.ExpectReply); notCheck != nil {
			res.Checks = append(res.Checks, *notCheck)
		}
	}

	judgeCheck := runLLMJudge(ctx, opts, judge, actualReply, res.DialogueSnapshot, intent)
	if judgeCheck != nil {
		res.Checks = append(res.Checks, *judgeCheck)
	}

	for _, c := range res.Checks {
		if !c.Passed {
			res.Passed = false
		}
	}
	res.Detail = summarizeChecks(res.Checks)
	res.Summary = buildSummary(res.Checks, actualReply)
	return res
}

func verifyReplyLengthCheck(reply string, minChars int) *EvalCheckResult {
	if minChars <= 0 {
		return nil
	}
	ok, detail := verifyReplyLength(reply, minChars)
	return &EvalCheckResult{
		Type: "reply_length", Passed: ok, Detail: detail,
		Expected: map[string]any{"min_reply_chars": minChars},
		Actual:   map[string]any{"length": len([]rune(strings.TrimSpace(reply)))},
	}
}

func verifyPassKeywordsCheck(reply string, keywords []string) *EvalCheckResult {
	if len(keywords) == 0 {
		return nil
	}
	ok, detail := verifyPassKeywords(reply, keywords)
	return &EvalCheckResult{
		Type: "keywords", Passed: ok, Detail: detail,
		Expected: map[string]any{"pass_keywords": keywords},
		Actual:   map[string]any{"reply_preview": truncateRunes(reply, 120)},
	}
}

func verifyMustCoverCheck(reply string, spec *ExpectReplySpec) *EvalCheckResult {
	if spec == nil || len(spec.MustCover) == 0 {
		return nil
	}
	ok, detail := verifyMustCover(reply, spec.MustCover)
	return &EvalCheckResult{
		Type: "must_cover", Passed: ok, Detail: detail,
		Expected: map[string]any{"must_cover": spec.MustCover},
		Actual:   map[string]any{"reply_preview": truncateRunes(reply, 120)},
	}
}

func verifyMustNotCheck(reply string, spec *ExpectReplySpec) *EvalCheckResult {
	if spec == nil || len(spec.MustNot) == 0 {
		return nil
	}
	ok, detail := verifyMustNot(reply, spec.MustNot)
	return &EvalCheckResult{
		Type: "must_not", Passed: ok, Detail: detail,
		Expected: map[string]any{"must_not": spec.MustNot},
		Actual:   map[string]any{"reply_preview": truncateRunes(reply, 120)},
	}
}

func runLLMJudge(ctx context.Context, opts TurnPlanCaseOptions, judge ReplyJudge, actualReply string, snapshot []DialogueSnapshotTurn, routing EvalCheckResult) *EvalCheckResult {
	if opts.ExpectReply == nil || strings.TrimSpace(opts.ExpectReply.Rubric) == "" {
		return nil
	}
	if opts.Judge == nil || !opts.Judge.Enabled {
		return nil
	}
	if strings.TrimSpace(actualReply) == "" {
		return &EvalCheckResult{
			Type: "llm_judge", Passed: false, Detail: "empty assistant reply",
			Expected: map[string]any{"rubric": opts.ExpectReply.Rubric},
		}
	}
	if judge == nil {
		return &EvalCheckResult{
			Type: "llm_judge", Passed: false, Detail: "judge provider unavailable",
			Expected: map[string]any{"rubric": opts.ExpectReply.Rubric},
		}
	}
	userMsg := lastUserMessage(opts.Dialogue, snapshot)
	out, err := judge.Judge(ctx, ReplyJudgeInput{
		UserMessage:     userMsg,
		DialogueContext: DialogueContextSummary(snapshot, 8),
		Rubric:          opts.ExpectReply.Rubric,
		ActualReply:     actualReply,
		RoutingSummary:  routing.Detail,
	})
	if err != nil {
		return &EvalCheckResult{
			Type: "llm_judge", Passed: false, Detail: "judge error: " + err.Error(),
			Expected: map[string]any{"rubric": opts.ExpectReply.Rubric},
		}
	}
	minScore := 0.7
	if opts.Judge.MinScore > 0 {
		minScore = opts.Judge.MinScore
	}
	passed := out.Pass && out.Score >= minScore
	score := out.Score
	return &EvalCheckResult{
		Type: "llm_judge", Passed: passed, Score: &score, Detail: out.Reason, Model: out.Model,
		Expected: map[string]any{"rubric": opts.ExpectReply.Rubric, "min_score": minScore},
		Actual: map[string]any{
			"score": out.Score, "pass": out.Pass, "gaps": out.Gaps,
			"reply_preview": truncateRunes(actualReply, 200),
		},
	}
}

func lastUserMessage(dialogue []EvalDialogueTurn, snapshot []DialogueSnapshotTurn) string {
	for i := len(dialogue) - 1; i >= 0; i-- {
		if dialogue[i].Role == "user" && strings.TrimSpace(dialogue[i].Text) != "" {
			return dialogue[i].Text
		}
	}
	for i := len(snapshot) - 1; i >= 0; i-- {
		if snapshot[i].Role == "user" && strings.TrimSpace(snapshot[i].Text) != "" {
			return snapshot[i].Text
		}
	}
	return ""
}

func summarizeChecks(checks []EvalCheckResult) string {
	if len(checks) == 0 {
		return "no checks"
	}
	var parts []string
	for _, c := range checks {
		tag := c.Type
		if c.Type == "" {
			tag = "check"
		}
		if c.Passed {
			parts = append(parts, tag+"=ok")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=fail(%s)", tag, c.Detail))
	}
	return strings.Join(parts, "; ")
}

func buildSummary(checks []EvalCheckResult, actualReply string) map[string]any {
	summary := map[string]any{
		"intent_pass":    false,
		"execution_pass": nil,
		"routing_pass":   false,
		"reply_pass":     true,
		"judge_pass":     nil,
		"judge_score":    nil,
		"judge_reason":   "",
	}
	for _, c := range checks {
		switch c.Type {
		case "intent":
			summary["intent_pass"] = c.Passed
			summary["routing_pass"] = c.Passed
		case "execution":
			summary["execution_pass"] = c.Passed
		case "routing":
			summary["routing_pass"] = c.Passed
		case "llm_judge":
			summary["judge_pass"] = c.Passed
			if c.Score != nil {
				summary["judge_score"] = *c.Score
			}
			summary["judge_reason"] = c.Detail
		default:
			if !c.Passed {
				summary["reply_pass"] = false
			}
		}
	}
	if actualReply != "" {
		summary["actual_reply_preview"] = truncateRunes(actualReply, 240)
	}
	return summary
}
