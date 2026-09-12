package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) registerEvalAutoClarifyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/dashboard/eval/auto-clarify", h.evalAutoClarify)
}

type evalAutoClarifyRequest struct {
	Question        string                  `json:"question"`
	Choices         []string                `json:"choices"`
	Dialogue        []eval.EvalDialogueTurn `json:"dialogue,omitempty"`
	ClarifyDefaults []string                `json:"clarify_defaults,omitempty"`
	SessionID       string                  `json:"session_id,omitempty"`
	UseLLM          bool                    `json:"use_llm"`
	ExpectIntent    *eval.ExpectIntentSpec  `json:"expect_intent,omitempty"`
	ExpectReply     *eval.ExpectReplySpec   `json:"expect_reply,omitempty"`
}

func (h *Handler) evalAutoClarify(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app unavailable")
		return
	}
	var req evalAutoClarifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	question := trimStrings(req.Question)
	choices := trimStringSlice(req.Choices)
	if question == "" {
		writeError(w, http.StatusBadRequest, "question required")
		return
	}
	hint := eval.ClarifyRecommendContext{
		Dialogue:        req.Dialogue,
		ClarifyDefaults: trimStringSlice(req.ClarifyDefaults),
		ExpectIntent:    req.ExpectIntent,
		ExpectReply:     req.ExpectReply,
	}
	if len(hint.Dialogue) == 0 && req.SessionID != "" {
		if store, err := h.App.SessionStore(); err == nil && store != nil {
			if chat, err := store.Load(req.SessionID); err == nil && chat != nil {
				hint.Dialogue = dialogueFromSession(chat)
			}
		}
	}
	var recommender eval.ClarifyRecommender
	if req.UseLLM {
		recommender = h.clarifyRecommender()
	}
	rec := eval.RecommendClarifyChoice(r.Context(), question, choices, hint, recommender)
	answer, ok := rec.AnswerChoice(choices)
	resp := map[string]any{
		"recommended_index":   rec.Index,
		"recommended_choice":  rec.Choice,
		"recommended_reason":  rec.Reason,
		"source":              rec.Source,
		"auto_pick_seconds":   rec.AutoPickSeconds,
		"answer":              answer,
		"answer_ok":           ok,
	}
	writeJSON(w, resp)
}

func (h *Handler) chatClarifyRecommend(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app unavailable")
		return
	}
	var req evalAutoClarifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	question := trimStrings(req.Question)
	choices := trimStringSlice(req.Choices)
	if question == "" || len(choices) == 0 {
		writeError(w, http.StatusBadRequest, "question and choices required")
		return
	}
	rec := h.recommendClarifyForSession(r.Context(), req.SessionID, question, choices, eval.ClarifyRecommendContext{
		Dialogue: req.Dialogue,
	})
	answer, ok := rec.AnswerChoice(choices)
	writeJSON(w, map[string]any{
		"recommended_index":  rec.Index,
		"recommended_choice": rec.Choice,
		"recommended_reason": rec.Reason,
		"source":             rec.Source,
		"auto_pick_seconds":  rec.AutoPickSeconds,
		"answer":             answer,
		"answer_ok":          ok,
	})
}

func trimStrings(s string) string {
	return strings.TrimSpace(s)
}

func trimStringSlice(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
