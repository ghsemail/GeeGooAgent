package report

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// chatSynthesis calls the LLM gateway with provider failover. Each provider is
// tried until validate accepts the response content or all providers are exhausted.
func (s *Synthesizer) chatSynthesis(
	ctx context.Context,
	messages []llm.Message,
	validate func(content string) error,
) (content string, model string, err error) {
	if s == nil || s.gateway == nil {
		return "", "", fmt.Errorf("synthesizer not available")
	}
	callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.ChatSynthesis(callCtx, messages, func(r *llm.Response) error {
		if r == nil {
			return fmt.Errorf("nil response")
		}
		body := strings.TrimSpace(r.Content)
		if body == "" {
			return fmt.Errorf("empty content")
		}
		if validate != nil {
			return validate(body)
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	model = strings.TrimSpace(resp.Usage.Model)
	if model == "" && s.gateway != nil {
		model = strings.TrimSpace(s.gateway.Model())
	}
	return strings.TrimSpace(resp.Content), model, nil
}
