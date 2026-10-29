package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const jevChoiceSystemPrompt = `你是 GeeGoo Agent 环内决策器（System-1）。只能从给定 choices 中选一项。
输入 JSON：question_id、kind、choices[{id,label}]、context。
只输出 JSON：{"choice_id":"…","confidence":0.0,"reason":"一句中文"}
- choice_id 必须是 choices 里某一项的 id
- 禁止输出 choices 以外的动作、工具参数或面向用户的成段回复`

// JEVChoiceHook calls the ops-configured decision LLM for typed multiple choice.
type JEVChoiceHook struct {
	Provider llm.Provider
	Timeout  time.Duration
}

func NewJEVChoiceHook(provider llm.Provider, timeout time.Duration) *JEVChoiceHook {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	return &JEVChoiceHook{Provider: provider, Timeout: timeout}
}

func (j *JEVChoiceHook) Decide(ctx context.Context, q Question) (Answer, error) {
	if j == nil || j.Provider == nil || len(q.Choices) == 0 {
		return Answer{}, fmt.Errorf("jev hook unavailable")
	}
	payload, err := json.Marshal(map[string]any{
		"question_id": q.ID,
		"kind":        q.Kind,
		"choices":     q.Choices,
		"context":     q.Context,
	})
	if err != nil {
		return Answer{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()
	resp, err := j.Provider.Chat(runCtx, []llm.Message{
		{Role: llm.RoleSystem, Content: jevChoiceSystemPrompt},
		{Role: llm.RoleUser, Content: string(payload)},
	}, nil, 0.05, 160)
	if err != nil {
		return Answer{}, err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return Answer{}, fmt.Errorf("jev empty response")
	}
	var raw struct {
		ChoiceID   string  `json:"choice_id"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}
	content := strings.TrimSpace(resp.Content)
	if i := strings.Index(content, "{"); i >= 0 {
		content = content[i:]
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return Answer{}, fmt.Errorf("jev parse: %w", err)
	}
	id := strings.TrimSpace(raw.ChoiceID)
	if !choiceAllowed(q.Choices, id) {
		return Answer{}, fmt.Errorf("jev choice %q not in options", id)
	}
	return Answer{
		QuestionID: q.ID,
		ChoiceID:   id,
		Confidence: raw.Confidence,
		Who:        WhoJev,
		Reason:     strings.TrimSpace(raw.Reason),
	}, nil
}

func choiceAllowed(choices []Choice, id string) bool {
	for _, c := range choices {
		if c.ID == id {
			return true
		}
	}
	return false
}
