package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// ReplyJudge scores assistant replies against a rubric (typically auxiliary LLM).
type ReplyJudge interface {
	Judge(ctx context.Context, in ReplyJudgeInput) (ReplyJudgeOutput, error)
}

// ReplyJudgeInput is passed to semantic judging.
type ReplyJudgeInput struct {
	UserMessage     string
	DialogueContext string
	Rubric          string
	ActualReply     string
	RoutingSummary  string
}

// ReplyJudgeOutput is structured judge output.
type ReplyJudgeOutput struct {
	Pass   bool     `json:"pass"`
	Score  float64  `json:"score"`
	Reason string   `json:"reason"`
	Gaps   []string `json:"gaps,omitempty"`
	Model  string   `json:"model,omitempty"`
}

// LLMReplyJudge uses an llm.Provider to score replies.
type LLMReplyJudge struct {
	Provider llm.Provider
	Policy   llm.Policy
}

func (j *LLMReplyJudge) Judge(ctx context.Context, in ReplyJudgeInput) (ReplyJudgeOutput, error) {
	if j == nil || j.Provider == nil {
		return ReplyJudgeOutput{}, fmt.Errorf("judge provider unavailable")
	}
	temp := 0.1
	maxTokens := 512
	if j.Policy != nil {
		dec := j.Policy.Decide(llm.Request{Kind: llm.TaskSynthesis})
		if dec.Temperature > 0 {
			temp = dec.Temperature
		}
		if dec.MaxTokens > 0 && dec.MaxTokens < maxTokens*4 {
			maxTokens = dec.MaxTokens
		}
	}
	system := `你是 GeeGoo Agent 评估裁判。根据「验收标准」判断助手回复是否满足用户意图。
只输出 JSON：{"pass":bool,"score":0~1,"reason":"一句话理由","gaps":["未覆盖点"]}
- pass=true 当 score>=0.7 且没有严重偏离意图
- 允许表述方式不同，关注业务意图是否达成
- 若应澄清而未澄清、或误触发错误能力，应判 fail`
	user := fmt.Sprintf(`## 对话上下文
%s

## 用户最后一轮
%s

## 路由结果
%s

## 验收标准
%s

## 助手实际回复
%s`,
		strings.TrimSpace(in.DialogueContext),
		strings.TrimSpace(in.UserMessage),
		strings.TrimSpace(in.RoutingSummary),
		strings.TrimSpace(in.Rubric),
		strings.TrimSpace(in.ActualReply),
	)
	resp, err := j.Provider.Chat(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: user},
	}, nil, temp, maxTokens)
	if err != nil {
		return ReplyJudgeOutput{}, err
	}
	out, err := parseJudgeJSON(resp.Content)
	if err != nil {
		return ReplyJudgeOutput{}, fmt.Errorf("parse judge: %w (raw=%q)", err, truncateRunes(resp.Content, 200))
	}
	out.Model = j.Provider.Model()
	if out.Score >= 0.7 && !out.Pass {
		out.Pass = true
	}
	return out, nil
}

func parseJudgeJSON(raw string) (ReplyJudgeOutput, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var out ReplyJudgeOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return ReplyJudgeOutput{}, err
	}
	if out.Score < 0 {
		out.Score = 0
	}
	if out.Score > 1 {
		out.Score = 1
	}
	return out, nil
}

func verifyMustCover(reply string, mustCover []string) (bool, string) {
	if len(mustCover) == 0 {
		return true, ""
	}
	lower := strings.ToLower(reply)
	var missing []string
	for _, kw := range mustCover {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if !strings.Contains(lower, strings.ToLower(kw)) {
			missing = append(missing, kw)
		}
	}
	if len(missing) > 0 {
		return false, "missing keywords: " + strings.Join(missing, ", ")
	}
	return true, "all must_cover keywords present"
}

func verifyMustNot(reply string, mustNot []string) (bool, string) {
	if len(mustNot) == 0 {
		return true, ""
	}
	lower := strings.ToLower(reply)
	for _, kw := range mustNot {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(kw)) {
			return false, "forbidden phrase present: " + kw
		}
	}
	return true, "no forbidden phrases"
}

func verifyReplyLength(reply string, minChars int) (bool, string) {
	if minChars <= 0 {
		return true, ""
	}
	n := len([]rune(strings.TrimSpace(reply)))
	if n < minChars {
		return false, fmt.Sprintf("reply length %d < min %d", n, minChars)
	}
	return true, fmt.Sprintf("reply length %d ok", n)
}

func verifyPassKeywords(reply string, keywords []string) (bool, string) {
	if len(keywords) == 0 {
		return true, ""
	}
	lower := strings.ToLower(reply)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
			return true, "matched keyword: " + kw
		}
	}
	return false, "no pass_keywords matched: " + strings.Join(keywords, " / ")
}
