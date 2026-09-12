package runtimeapi

import (
	"encoding/json"
	"net/http"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) registerEvalSuggestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/dashboard/eval/suggest-expectations", h.evalSuggestExpectations)
}

type evalSuggestRequest struct {
	Dialogue []eval.EvalDialogueTurn `json:"dialogue"`
	UseLLM   bool                    `json:"use_llm"`
}

func (h *Handler) evalSuggestExpectations(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app unavailable")
		return
	}
	var req evalSuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Dialogue) == 0 {
		writeError(w, http.StatusBadRequest, "dialogue required")
		return
	}
	var replySuggester eval.ExpectReplySuggester
	if req.UseLLM {
		if provider := h.App.OpsBackgroundProvider(); provider != nil {
			replySuggester = &eval.LLMExpectReplySuggester{
				Provider: provider,
				Policy:   h.App.OpsBackgroundPolicy(),
			}
		}
	}
	suggested, err := eval.SuggestCaseExpectations(r.Context(), req.Dialogue, h.App.IntentPlanner(), replySuggester)
	resp := map[string]any{"suggestions": suggested}
	if err != nil {
		resp["llm_warning"] = err.Error()
	}
	writeJSON(w, resp)
}
