package eval

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// TurnPlanCaseOptions is one dashboard eval case (live chat execution).
type TurnPlanCaseOptions struct {
	Category       string   `json:"category"`
	PlanOnly       bool     `json:"plan_only"`
	SessionCleanup string   `json:"session_cleanup"`
	DualModelEval  bool     `json:"dual_model_eval"`
	Message        string   `json:"message"`
	SetupMessages  []string `json:"setup_messages,omitempty"`
	ExpectDomain   string   `json:"expect_domain"`
	ExpectMode     string   `json:"expect_mode"`
	ExpectSOP      bool     `json:"expect_sop"`
	ForbidTools    []string `json:"forbid_tools,omitempty"`
	RequireTools   []string `json:"require_tools,omitempty"`
	PassKeywords   []string `json:"pass_keywords,omitempty"`
	MinReplyChars  int      `json:"min_reply_chars,omitempty"`
	TurnID         string   `json:"turn_id,omitempty"`
	Dialogue       []EvalDialogueTurn `json:"dialogue,omitempty"`
	ExpectReply    *ExpectReplySpec   `json:"expect_reply,omitempty"`
	ExpectRouting  *ExpectRoutingSpec `json:"expect_routing,omitempty"`
	Judge          *EvalJudgeConfig   `json:"judge,omitempty"`
}

// TurnPlanEvalCaseDef is one seeded dashboard eval row.
type TurnPlanEvalCaseDef struct {
	ID          string
	Title       string
	Description string
	Steps       []string
	SortOrder   int
	Options     TurnPlanCaseOptions
}

// ParseTurnPlanCaseOptions decodes options_json for a live turn_plan eval case.
func ParseTurnPlanCaseOptions(raw []byte) (TurnPlanCaseOptions, error) {
	var opts TurnPlanCaseOptions
	if err := json.Unmarshal(raw, &opts); err != nil {
		return TurnPlanCaseOptions{}, err
	}
	if opts.Category == "" {
		opts.Category = "turn_plan"
	}
	return opts, nil
}

// IndividualTurnPlanEvalCases returns live eval cases (one session each).
func IndividualTurnPlanEvalCases() []TurnPlanEvalCaseDef {
	live := defaultTurnPlanLiveCases()
	out := make([]TurnPlanEvalCaseDef, 0, len(live))
	for i, c := range live {
		opts := TurnPlanCaseOptions{
			Category:       "turn_plan",
			PlanOnly:       false,
			SessionCleanup: "before_run",
			DualModelEval:  false,
			Message:        c.Message,
			SetupMessages:  append([]string(nil), c.SetupMessages...),
			ExpectDomain:   c.ExpectDomain,
			ExpectMode:     c.ExpectMode,
			ExpectSOP:      c.ExpectSOP,
			ForbidTools:    append([]string(nil), c.ForbidTools...),
			RequireTools:   append([]string(nil), c.RequireTools...),
			MinReplyChars:  20,
			TurnID:         c.ID,
			ExpectRouting: &ExpectRoutingSpec{
				Domain: c.ExpectDomain, Mode: c.ExpectMode, SOP: c.ExpectSOP,
				RequireTools: append([]string(nil), c.RequireTools...),
				ForbidTools:  append([]string(nil), c.ForbidTools...),
			},
		}
		opts = opts.Normalize()
		out = append(out, TurnPlanEvalCaseDef{
			ID:          "turn_plan_" + c.ID,
			Title:       "TurnPlan · " + c.Title,
			Description: c.Description,
			Steps:       liveCaseSteps(c),
			SortOrder:   50 + i,
			Options:     opts,
		})
	}
	return out
}

func liveCaseSteps(c TurnPlanLiveCase) []string {
	return []string{
		"新 session：运行前清空 Dock Chat",
		liveDialogueStep(c.SetupMessages, c.Message),
		"verify：校验路由/工具/回复关键词 + LLM 语义评判",
	}
}

func liveDialogueStep(setup []string, message string) string {
	if len(setup) == 0 {
		return fmt.Sprintf("单轮发送：「%s」", message)
	}
	parts := append(append([]string(nil), setup...), message)
	var b strings.Builder
	b.WriteString("同 session 按序发送：")
	for i, part := range parts {
		if i > 0 {
			b.WriteString(" → ")
		}
		b.WriteString("「")
		b.WriteString(part)
		b.WriteString("」")
	}
	return b.String()
}

// TurnPlanSnapshot is persisted on the chat session after each agent turn.
type TurnPlanSnapshot struct {
	Domain     string   `json:"domain"`
	Mode       string   `json:"mode"`
	SOP        bool     `json:"sop"`
	ToolsAllow []string `json:"tools_allow,omitempty"`
}

// VerifyTurnPlanLive checks a completed chat turn against case expectations.
func VerifyTurnPlanLive(chat *chatsession.ChatSession, opts TurnPlanCaseOptions) TurnPlanResult {
	turnID := opts.TurnID
	if turnID == "" {
		turnID = "live"
	}
	res := TurnPlanResult{TurnID: turnID, Passed: true}
	if chat == nil {
		res.Passed = false
		res.Detail = "session not found"
		return res
	}

	snap, ok := chatsession.LastTurnPlanFromSession(chat)
	if !ok {
		res.Passed = false
		res.Detail = "missing last_turn_plan on session (turn did not complete?)"
		return res
	}

	var problems []string
	if snap.Domain != opts.ExpectDomain {
		problems = append(problems, fmt.Sprintf("domain=%s want %s", snap.Domain, opts.ExpectDomain))
	}
	if snap.Mode != opts.ExpectMode {
		problems = append(problems, fmt.Sprintf("mode=%s want %s", snap.Mode, opts.ExpectMode))
	}
	if snap.SOP != opts.ExpectSOP {
		problems = append(problems, fmt.Sprintf("sop=%v want %v", snap.SOP, opts.ExpectSOP))
	}

	called := chatsession.LastTurnToolsCalledFromSession(chat)
	for _, tool := range opts.ForbidTools {
		if containsString(called, tool) {
			problems = append(problems, fmt.Sprintf("forbid tool %s called", tool))
		}
	}
	for _, tool := range opts.RequireTools {
		if !containsString(called, tool) {
			problems = append(problems, fmt.Sprintf("missing tool call %s", tool))
		}
	}

	if len(problems) > 0 {
		res.Passed = false
		res.Detail = strings.Join(problems, "; ")
		return res
	}
	res.Detail = fmt.Sprintf("%s/%s sop=%v tools=%s", snap.Domain, snap.Mode, snap.SOP, strings.Join(called, ","))
	return res
}

func containsString(list []string, name string) bool {
	for _, t := range list {
		if t == name {
			return true
		}
	}
	return false
}
