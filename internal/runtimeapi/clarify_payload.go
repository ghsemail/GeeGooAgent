package runtimeapi

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func pendingClarifyPayload(p PendingClarify) map[string]any {
	out := map[string]any{
		"session_id": p.SessionID,
		"question":   p.Question,
		"choices":    p.Choices,
	}
	if p.RecommendedIndex >= 0 && len(p.Choices) > 0 && p.RecommendedIndex < len(p.Choices) {
		out["recommended_index"] = p.RecommendedIndex
		out["recommended_choice"] = p.Choices[p.RecommendedIndex]
	}
	if reason := strings.TrimSpace(p.RecommendedReason); reason != "" {
		out["recommended_reason"] = reason
	}
	secs := p.AutoPickSeconds
	if secs <= 0 {
		secs = eval.DefaultClarifyAutoPickSeconds
	}
	out["auto_pick_seconds"] = secs
	return out
}

func (h *Handler) recommendClarifyForSession(
	ctx context.Context,
	sessionID, question string,
	choices []string,
	hint eval.ClarifyRecommendContext,
) eval.ClarifyRecommendation {
	if len(hint.Dialogue) == 0 && sessionID != "" && h.App != nil {
		if store, err := h.App.SessionStore(); err == nil && store != nil {
			if chat, err := store.Load(sessionID); err == nil && chat != nil {
				hint.Dialogue = dialogueFromSession(chat)
			}
		}
	}
	rec := eval.RecommendClarifyChoice(ctx, question, choices, hint, h.clarifyRecommender())
	if rec.AutoPickSeconds <= 0 {
		rec.AutoPickSeconds = eval.DefaultClarifyAutoPickSeconds
	}
	if rec.Choice == "" {
		if answer, ok := rec.AnswerChoice(choices); ok {
			rec.Choice = answer
		}
	}
	return rec
}

func (h *Handler) clarifyRecommender() eval.ClarifyRecommender {
	if h == nil || h.App == nil {
		return nil
	}
	provider := h.App.OpsBackgroundProvider()
	if provider == nil {
		return nil
	}
	return &eval.LLMClarifyRecommender{
		Provider: provider,
		Policy:   h.App.OpsBackgroundPolicy(),
	}
}

func dialogueFromSession(chat *chatsession.ChatSession) []eval.EvalDialogueTurn {
	if chat == nil {
		return nil
	}
	out := make([]eval.EvalDialogueTurn, 0, len(chat.Messages))
	for _, msg := range chat.Messages {
		if msg.Role != llm.RoleUser {
			continue
		}
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			continue
		}
		out = append(out, eval.EvalDialogueTurn{Role: "user", Text: text})
	}
	return out
}

func pendingFromRecommendation(sessionID, question string, choices []string, rec eval.ClarifyRecommendation) PendingClarify {
	idx := rec.Index
	if answer, ok := rec.AnswerChoice(choices); ok {
		idx = indexOfString(choices, answer)
	}
	return PendingClarify{
		SessionID:         sessionID,
		Question:          question,
		Choices:           append([]string(nil), choices...),
		RecommendedIndex:  idx,
		RecommendedReason: rec.Reason,
		AutoPickSeconds:   rec.AutoPickSeconds,
	}
}

func indexOfString(items []string, target string) int {
	for i, s := range items {
		if s == target {
			return i
		}
	}
	return 0
}

func (h *Handler) waitClarifyWithAutoPick(
	ctx context.Context,
	sessionID, question string,
	choices []string,
	meta PendingClarify,
	onPending func(PendingClarify),
) (string, bool) {
	if h == nil || h.clarify == nil {
		return "", false
	}
	secs := meta.AutoPickSeconds
	if secs <= 0 {
		secs = eval.DefaultClarifyAutoPickSeconds
	}
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type waitResult struct {
		answer string
		ok     bool
	}
	done := make(chan waitResult, 1)
	go func() {
		answer, ok := h.clarify.WaitWithRecommend(waitCtx, sessionID, question, choices, meta, onPending)
		done <- waitResult{answer: answer, ok: ok}
	}()

	timer := time.NewTimer(time.Duration(secs) * time.Second)
	defer timer.Stop()

	select {
	case res := <-done:
		return res.answer, res.ok
	case <-timer.C:
		answer, ok := recommendAnswerFromPending(meta)
		if !ok {
			if auto, picked := tools.AutoClarifyChoice(question, choices); picked {
				answer = auto
			} else {
				return "", false
			}
		}
		_ = h.clarify.Answer(sessionID, answer, true)
		select {
		case res := <-done:
			if res.ok && strings.TrimSpace(res.answer) != "" {
				return res.answer, true
			}
			return answer, true
		case <-waitCtx.Done():
			return answer, true
		case <-time.After(2 * time.Second):
			return answer, true
		}
	case <-ctx.Done():
		cancel()
		return "", false
	}
}

func recommendAnswerFromPending(meta PendingClarify) (string, bool) {
	rec := eval.ClarifyRecommendation{
		Index:  meta.RecommendedIndex,
		Reason: meta.RecommendedReason,
	}
	if meta.RecommendedIndex >= 0 && meta.RecommendedIndex < len(meta.Choices) {
		rec.Choice = meta.Choices[meta.RecommendedIndex]
	}
	return rec.AnswerChoice(meta.Choices)
}
