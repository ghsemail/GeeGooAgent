package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/lab"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func (h *Handler) registerSignalDiagnoseLabRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/signal-diagnose/lab-detail", h.signalDiagnoseLabDetail)
}

type signalDiagnoseLabDetailBody struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
	DetailRef string `json:"detail_ref"`
	MCPToken  string `json:"mcp_token"`
}

func (h *Handler) signalDiagnoseLabDetail(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	var body signalDiagnoseLabDetailBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if !requireMCPTokenForChat(w, r, body.MCPToken, h.configMCPToken()) {
		return
	}
	sessionID := strings.TrimSpace(body.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required")
		return
	}
	ref := strings.TrimSpace(body.DetailRef)
	if ref == "" {
		writeError(w, http.StatusBadRequest, "detail_ref required")
		return
	}
	runID := strings.TrimSpace(body.RunID)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id required")
		return
	}
	chat, err := store.Load(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	userID := resolveUserID(r)
	if !enforceSessionAccess(w, chat, userID) {
		return
	}
	flow := workflowchat.ResolveFlowForLabDetail(chat, runID)
	if flow == nil {
		writeError(w, http.StatusNotFound, "lab diagnose snapshot not found or expired")
		return
	}
	if !workflowchat.FlowSupportsLabDetail(flow) {
		writeError(w, http.StatusBadRequest, "diagnose run has no probe bars for lab detail")
		return
	}
	if cached, ok := workflowchat.CachedLabEpisodeDetail(chat, runID, ref); ok {
		strategydev.PrepareLabEpisodeDetailForEmit(&cached)
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"run_id": flow.RunID,
			"detail": cached,
			"cached": true,
		}
		if hint := strategydev.LabDetailRefMismatchHint(flow.LabEpisodeSummaries, ref); hint != "" {
			payload["ref_hint"] = hint
		}
		_ = json.NewEncoder(w).Encode(payload)
		return
	}
	bars, _ := flow.ProbeRaw["bars"].([]any)
	lookback, _ := flow.ProbeRaw["bars_lookback"].([]any)
	freq := "60m"
	if len(flow.Strategies) > 0 && strings.TrimSpace(flow.Strategies[0].Frequency) != "" {
		freq = flow.Strategies[0].Frequency
	}
	months := flow.MonthsBack
	if months <= 0 {
		months = 3
	}
	toolCtx := h.App.ToolContextWithContext(r.Context(), chat.ID)
	toolCtx.UserID = userID
	toolCtx.MCPToken = resolveChatMCPToken(r, body.MCPToken, h.configMCPToken())
	runner := func(ctx context.Context, name string, args map[string]any) tools.Result {
		return h.App.Registry.Execute(tools.CallRequest{Name: name, Arguments: args}, toolCtx)
	}
	cfg := h.App.Config
	labHTTP := signalclient.NewHTTPClient(cfg.LabAPIURL(), cfg.LabAPIKey())
	fetcher := lab.SignalBarsFetcher{Client: labHTTP}
	jev := h.App.NewStrategyDevJevClient()
	detail, err := strategydev.BuildLabEpisodeDetail(r.Context(), strategydev.LabDetailInput{
		Run: strategydev.ToolRunner(runner), Code: flow.StockCode, SignalFreq: freq, Months: months,
		KeyLevels: keyLevelBandsFromFlowSnapshot(flow), ProbeRaw: flow.ProbeRaw, Bars: bars,
		LookbackBars: lookback, LabFetcher: fetcher, TAClient: labHTTP,
		DetailTimeout: strategydev.DefaultLabDetailPhaseTimeout,
		PATimeout:     strategydev.DefaultLabDetailPATimeout,
		TATimeout:     strategydev.DefaultLabDetailTATimeout,
		JEVTimeout:    strategydev.DefaultLabDetailJEVTimeout,
		CachedMTF: &flow.LabKlineMTF, SessionID: sessionID, RunID: flow.RunID, Jev: jev,
		LongOnly: flow.TradeDirection.LongOnlyEpisodeEval(),
	}, ref)
	if err != nil {
		if strategydev.IsBadLabRef(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	workflowchat.StoreLabEpisodeDetail(chat, nil, flow.RunID, ref, detail)
	if err := store.Save(chat); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]any{
		"run_id": flow.RunID,
		"detail": detail,
		"cached": false,
	}
	if hint := strategydev.LabDetailRefMismatchHint(flow.LabEpisodeSummaries, ref); hint != "" {
		payload["ref_hint"] = hint
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func keyLevelBandsFromFlowSnapshot(flow *workflowchat.Flow) strategydev.KeyLevelBands {
	if flow == nil {
		return strategydev.KeyLevelBands{}
	}
	kl := flow.KeyLevels
	return strategydev.KeyLevelBands{
		SupportLow: kl.SupportLow, SupportHigh: kl.SupportHigh,
		ResistLow: kl.ResistLow, ResistHigh: kl.ResistHigh,
	}
}
