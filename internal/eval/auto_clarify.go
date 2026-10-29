package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clarifycontext"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const DefaultClarifyAutoPickSeconds = 20

// ClarifyAutoPickGraceSeconds is extra wait after auto-pick before unblocking the agent turn.
const ClarifyAutoPickGraceSeconds = 3

// JEVClarifyRecommendTimeout is the hard cap for JEV clarify recommendation calls.
const JEVClarifyRecommendTimeout = 2500 * time.Millisecond

// EvalClarifyWaitBudgetSec is per-clarify UI wait budget (JEV + auto-pick + grace) for eval timeouts.
const EvalClarifyWaitBudgetSec = DefaultClarifyAutoPickSeconds + 3 + ClarifyAutoPickGraceSeconds

// ClarifyRecommendContext supplies optional dialogue / script hints for auto-answer.
type ClarifyRecommendContext struct {
	Dialogue        []EvalDialogueTurn
	ClarifyDefaults []string
	ExpectIntent    *ExpectIntentSpec
	ExpectReply     *ExpectReplySpec
	// ClarifyPreference is eval-case clarify_reply (soft hint for JEV, not script-first pick).
	ClarifyPreference string
	// Bundle is structured session summary + slots for JEV (built from ChatSession when available).
	Bundle *clarifycontext.Bundle
}

// ClarifyRecommendation is the suggested clarify choice for UI and timeout auto-pick.
type ClarifyRecommendation struct {
	Index           int    `json:"recommended_index"`
	Choice          string `json:"recommended_choice,omitempty"`
	Reason          string `json:"recommended_reason,omitempty"`
	Source          string `json:"source,omitempty"`
	AutoPickSeconds int    `json:"auto_pick_seconds"`
}

// ClarifyRecommender optionally uses an LLM to pick among clarify choices.
type ClarifyRecommender interface {
	Recommend(
		ctx context.Context,
		question string,
		choices []string,
		hint ClarifyRecommendContext,
	) (ClarifyRecommendation, error)
}

// LLMClarifyRecommender picks a clarify option from dialogue + rubric context.
type LLMClarifyRecommender struct {
	Provider llm.Provider
	Policy   llm.Policy
}

func (g *LLMClarifyRecommender) Recommend(
	ctx context.Context,
	question string,
	choices []string,
	hint ClarifyRecommendContext,
) (ClarifyRecommendation, error) {
	if g == nil || g.Provider == nil || len(choices) == 0 {
		return ClarifyRecommendation{}, fmt.Errorf("llm recommender unavailable")
	}
	temp := 0.1
	maxTokens := 256
	if g.Policy != nil {
		dec := g.Policy.Decide(llm.Request{Kind: llm.TaskSynthesis})
		if dec.Temperature > 0 {
			temp = dec.Temperature
		}
		if dec.MaxTokens > 0 && dec.MaxTokens < maxTokens*4 {
			maxTokens = dec.MaxTokens
		}
	}
	system := `你是 GeeGoo Agent 澄清助手。用户在对话中被要求从若干选项中做选择。
只输出 JSON：{"index":0,"reason":"…"}
- index 必须是 0 到 N-1 的整数（对应下方列出的选项序号，从 0 起）
- reason 用一句中文说明为何推荐该选项`
	user := formatClarifyRecommendPrompt(question, choices, hint)
	resp, err := g.Provider.Chat(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: user},
	}, nil, temp, maxTokens)
	if err != nil {
		return ClarifyRecommendation{}, err
	}
	rec, err := parseClarifyRecommendJSON(resp.Content, choices)
	if err != nil {
		return ClarifyRecommendation{}, err
	}
	rec.Source = "llm"
	rec.AutoPickSeconds = DefaultClarifyAutoPickSeconds
	return rec, nil
}

func formatClarifyRecommendPrompt(question string, choices []string, hint ClarifyRecommendContext) string {
	var b strings.Builder
	b.WriteString("## 澄清问题\n")
	b.WriteString(strings.TrimSpace(question))
	b.WriteString("\n\n## 选项\n")
	for i, c := range choices {
		fmt.Fprintf(&b, "%d. %s\n", i, c)
	}
	if len(hint.Dialogue) > 0 {
		b.WriteString("\n## 对话上下文\n")
		for _, turn := range hint.Dialogue {
			if turn.Role != "user" {
				continue
			}
			text := strings.TrimSpace(turn.Text)
			if text == "" {
				continue
			}
			if turn.OnClarify {
				fmt.Fprintf(&b, "- [澄清默认] %s\n", text)
			} else {
				fmt.Fprintf(&b, "- %s\n", text)
			}
		}
	}
	if intent := hint.ExpectIntent; intent != nil {
		fmt.Fprintf(&b, "\n## 期望路由\ndomain=%s mode=%s act=%s\n", intent.Domain, intent.Mode, intent.Act)
	}
	if reply := hint.ExpectReply; reply != nil {
		if rubric := strings.TrimSpace(reply.Rubric); rubric != "" {
			fmt.Fprintf(&b, "\n## 验收标准\n%s\n", rubric)
		}
	}
	if pref := strings.TrimSpace(hint.ClarifyPreference); pref != "" {
		fmt.Fprintf(&b, "\n## Eval 软偏好（clarify_reply）\n%s\n", pref)
	}
	if hint.Bundle != nil {
		if s := strings.TrimSpace(hint.Bundle.SessionSummary); s != "" {
			fmt.Fprintf(&b, "\n## 会话摘要\n%s\n", s)
		}
		if s := strings.TrimSpace(hint.Bundle.WorkingState); s != "" {
			fmt.Fprintf(&b, "\n## 任务态\n%s\n", s)
		}
		if len(hint.Bundle.Dialogue) > 0 {
			b.WriteString("\n## 近期对话摘要\n")
			for _, line := range hint.Bundle.Dialogue {
				fmt.Fprintf(&b, "- %s: %s\n", line.Role, line.Summary)
			}
		}
		if len(hint.Bundle.RecentSteps) > 0 {
			b.WriteString("\n## 近期工具\n")
			for _, line := range hint.Bundle.RecentSteps {
				fmt.Fprintf(&b, "- %s\n", line)
			}
		}
	}
	return b.String()
}

func parseClarifyRecommendJSON(raw string, choices []string) (ClarifyRecommendation, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var parsed struct {
		Index  int    `json:"index"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return ClarifyRecommendation{}, err
	}
	if parsed.Index < 0 || parsed.Index >= len(choices) {
		return ClarifyRecommendation{}, fmt.Errorf("index out of range: %d", parsed.Index)
	}
	return ClarifyRecommendation{
		Index:  parsed.Index,
		Choice: choices[parsed.Index],
		Reason: strings.TrimSpace(parsed.Reason),
	}, nil
}

// RecommendClarifyChoice picks a recommended option: JEV/LLM chain → heuristic.
// ClarifyDefaults and ClarifyPreference are soft hints for recommenders, not script-first picks.
func RecommendClarifyChoice(
	ctx context.Context,
	question string,
	choices []string,
	hint ClarifyRecommendContext,
	recommender ClarifyRecommender,
) ClarifyRecommendation {
	hint = enrichClarifyRecommendHint(hint)
	if recommender != nil && len(choices) > 0 {
		if rec, err := recommender.Recommend(ctx, question, choices, hint); err == nil && rec.valid(choices) {
			if rec.AutoPickSeconds <= 0 {
				rec.AutoPickSeconds = DefaultClarifyAutoPickSeconds
			}
			return rec
		}
	}
	if answer, ok := tools.AutoClarifyChoice(question, choices); ok {
		idx := indexOfChoice(choices, answer)
		return ClarifyRecommendation{
			Index:           idx,
			Choice:          answer,
			Reason:          "启发式默认",
			Source:          "heuristic",
			AutoPickSeconds: DefaultClarifyAutoPickSeconds,
		}
	}
	return ClarifyRecommendation{AutoPickSeconds: DefaultClarifyAutoPickSeconds}
}

func enrichClarifyRecommendHint(hint ClarifyRecommendContext) ClarifyRecommendContext {
	if strings.TrimSpace(hint.ClarifyPreference) == "" {
		defaults := nonEmptyStrings(hint.ClarifyDefaults)
		if len(defaults) > 0 {
			hint.ClarifyPreference = defaults[0]
		}
	}
	for _, def := range nonEmptyStrings(hint.ClarifyDefaults) {
		seen := false
		for _, turn := range hint.Dialogue {
			if turn.OnClarify && strings.TrimSpace(turn.Text) == def {
				seen = true
				break
			}
		}
		if !seen {
			hint.Dialogue = append(hint.Dialogue, EvalDialogueTurn{
				Role:      "user",
				Text:      def,
				OnClarify: true,
			})
		}
	}
	return hint
}

// ClarifyTurnWaitPadding returns extra seconds eval runners should allow per turn that may clarify.
func ClarifyTurnWaitPadding(opts TurnPlanCaseOptions) int {
	opts = opts.Normalize()
	if opts.AutoClarifyOnly {
		return EvalClarifyWaitBudgetSec
	}
	if strings.TrimSpace(opts.ClarifyReply) != "" {
		return EvalClarifyWaitBudgetSec
	}
	_, clarify := DialogueExecutionPlan(opts)
	if len(clarify) > 0 {
		return EvalClarifyWaitBudgetSec * len(clarify)
	}
	if strings.EqualFold(opts.intent().Mode, "clarify") {
		return EvalClarifyWaitBudgetSec
	}
	return 0
}

func (r ClarifyRecommendation) valid(choices []string) bool {
	if len(choices) == 0 {
		return false
	}
	if r.Index >= 0 && r.Index < len(choices) {
		return true
	}
	if r.Choice != "" {
		for _, c := range choices {
			if c == r.Choice {
				return true
			}
		}
	}
	return false
}

// AnswerChoice returns the choice string to submit, normalizing index/choice fields.
func (r ClarifyRecommendation) AnswerChoice(choices []string) (string, bool) {
	if len(choices) == 0 {
		return "", false
	}
	if r.Index >= 0 && r.Index < len(choices) {
		return choices[r.Index], true
	}
	if r.Choice != "" {
		for _, c := range choices {
			if c == r.Choice {
				return c, true
			}
		}
	}
	return "", false
}

func indexOfChoice(choices []string, answer string) int {
	for i, c := range choices {
		if c == answer {
			return i
		}
	}
	return 0
}
