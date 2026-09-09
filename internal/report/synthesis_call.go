package report

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const (
	defaultSynthesisRetryInterval = 5 * time.Minute
	defaultSynthesisMaxRetries    = 3
)

type synthesisRetryPolicy struct {
	interval   time.Duration
	maxRetries int
}

// synthesisSleep waits between LLM synthesis retries; replaced in tests.
var synthesisSleep = sleepWithContext

var (
	synthesisRetryInterval = defaultSynthesisRetryInterval
	synthesisMaxRetries    = defaultSynthesisMaxRetries
	synthesisRetryOverride *synthesisRetryPolicy
)

// SetSynthesisRetryPolicyForTest overrides retry interval and max retries (package report tests only).
func SetSynthesisRetryPolicyForTest(interval time.Duration, maxRetries int) func() {
	prev := synthesisRetryOverride
	synthesisRetryOverride = &synthesisRetryPolicy{interval: interval, maxRetries: maxRetries}
	return func() { synthesisRetryOverride = prev }
}

func effectiveSynthesisRetryPolicy() (interval time.Duration, maxRetries int) {
	if synthesisRetryOverride != nil {
		return synthesisRetryOverride.interval, synthesisRetryOverride.maxRetries
	}
	if testing.Testing() {
		return 0, 0
	}
	return synthesisRetryInterval, synthesisMaxRetries
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// chatSynthesis calls the LLM gateway with provider failover. On failure it retries
// every synthesisRetryInterval up to synthesisMaxRetries times, logging each attempt.
func (s *Synthesizer) chatSynthesis(
	ctx context.Context,
	messages []llm.Message,
	validate func(content string) error,
) (content string, model string, err error) {
	if s == nil || s.gateway == nil {
		return "", "", fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	retryWait, maxRetries := effectiveSynthesisRetryPolicy()
	if maxRetries < 0 {
		maxRetries = 0
	}
	if retryWait < 0 {
		retryWait = 0
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		body, usedModel, callErr := s.chatSynthesisOnce(ctx, messages, validate)
		if callErr == nil {
			if attempt > 0 {
				slog.Info("llm synthesis succeeded after retries",
					"retry_count", attempt,
					"max_retries", maxRetries,
					"model", usedModel,
				)
			}
			return body, usedModel, nil
		}
		lastErr = callErr
		if attempt >= maxRetries {
			break
		}
		slog.Warn("llm synthesis failed, will retry",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"retry_in", retryWait.String(),
			"error", callErr,
		)
		if err := synthesisSleep(ctx, retryWait); err != nil {
			return "", "", fmt.Errorf("llm synthesis retry cancelled after %d attempt(s): %w", attempt+1, err)
		}
		slog.Info("llm synthesis retrying",
			"retry", attempt+1,
			"max_retries", maxRetries,
		)
	}
	if maxRetries == 0 {
		return "", "", lastErr
	}
	return "", "", fmt.Errorf("llm synthesis failed after %d retries: %w", maxRetries, lastErr)
}

func (s *Synthesizer) chatSynthesisOnce(
	ctx context.Context,
	messages []llm.Message,
	validate func(content string) error,
) (content string, model string, err error) {
	attemptCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.timeout)
	defer cancel()

	callCtx := llm.WithCallMeta(attemptCtx, llm.CallMeta{Kind: llm.TaskSynthesis})
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
