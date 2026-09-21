package eval

import (
	"context"
	"fmt"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clarifycontext"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const jevClarifySystemPrompt = `你是 GeeGoo 澄清决策器（System-1）。用户必须从给定 options 中选一项以继续当前任务。
输入是 JSON，含 session_summary、dialogue、last_turn、slots 等上下文以及 question/options。
只输出 JSON：{"index":0,"reason":"一句中文","confidence":0.0}
- index 必须是 0 到 N-1 的整数（对应 options 里的 index）
- reason 给用户看，简短说明为何选这项
- 禁止输出 options 以外的动作或自由文本`

// JEVClarifyRecommender uses the ops-configured Jev decision model.
type JEVClarifyRecommender struct {
	Provider llm.Provider
	Timeout  time.Duration
}

// NewJEVClarifyRecommender wraps a decision LLM with a hard timeout.
func NewJEVClarifyRecommender(provider llm.Provider, timeout time.Duration) *JEVClarifyRecommender {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	return &JEVClarifyRecommender{Provider: provider, Timeout: timeout}
}

func (j *JEVClarifyRecommender) Recommend(
	ctx context.Context,
	question string,
	choices []string,
	hint ClarifyRecommendContext,
) (ClarifyRecommendation, error) {
	if j == nil || j.Provider == nil || len(choices) == 0 {
		return ClarifyRecommendation{}, fmt.Errorf("jev recommender unavailable")
	}
	bundle := hint.bundleForDecision(question, choices)
	userJSON, err := bundle.DecisionJSON()
	if err != nil {
		return ClarifyRecommendation{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()
	resp, err := j.Provider.Chat(runCtx, []llm.Message{
		{Role: llm.RoleSystem, Content: jevClarifySystemPrompt},
		{Role: llm.RoleUser, Content: userJSON},
	}, nil, 0.05, 160)
	if err != nil {
		return ClarifyRecommendation{}, err
	}
	if resp == nil {
		return ClarifyRecommendation{}, fmt.Errorf("jev empty response")
	}
	rec, err := parseClarifyRecommendJSON(resp.Content, choices)
	if err != nil {
		return ClarifyRecommendation{}, err
	}
	rec.Source = "jev"
	rec.AutoPickSeconds = DefaultClarifyAutoPickSeconds
	return rec, nil
}

func (hint ClarifyRecommendContext) bundleForDecision(question string, choices []string) clarifycontext.Bundle {
	if hint.Bundle != nil {
		return *hint.Bundle
	}
	b := clarifycontext.BuildFromChat(nil, question, choices, "eval")
	if len(hint.Dialogue) > 0 {
		lines := make([]clarifycontext.DialogueLine, 0, len(hint.Dialogue))
		for _, turn := range hint.Dialogue {
			text := turn.Text
			if turn.Role != "user" {
				continue
			}
			if text == "" {
				continue
			}
			lines = append(lines, clarifycontext.DialogueLine{Role: turn.Role, Summary: text})
		}
		b.Dialogue = lines
	}
	return b
}
