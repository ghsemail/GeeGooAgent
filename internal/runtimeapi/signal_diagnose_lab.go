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
	// Rebuild bypasses session lab-detail cache (e.g. macro backfill after JEV logic changes).
	Rebuild bool `json:"rebuild"`
	// KlineJEVProfile e.g. daily_macro for macro monitor (daily patterns + three-bar OHLC only).
	KlineJEVProfile string `json:"kline_jev_profile,omitempty"`
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
	if body.Rebuild {
		strategydev.ClearCachedLabKlineMTF(sessionID, flow.RunID)
		flow.LabKlineMTF = nil
	}
	wantProfile := strings.TrimSpace(body.KlineJEVProfile)
	cacheRef := strategydev.LabDetailCacheRef(ref, wantProfile)
	if !body.Rebuild {
		if cached, ok := workflowchat.CachedLabEpisodeDetail(chat, runID, cacheRef); ok &&
			strategydev.LabDetailProfileMatches(cached, wantProfile) {
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
	}
	strategydev.EnsureProbeChartBarsForLab(flow.ProbeRaw, flow.MonthsBack)
	bars := strategydev.LabDetailChartBars(flow.ProbeRaw)
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
		LookbackBars: lookback, SignalEval: flow.SignalEval, LabFetcher: fetcher, TAClient: labHTTP, IndexTrendClient: labHTTP,
		DetailTimeout: strategydev.DefaultLabDetailPhaseTimeout,
		PATimeout:     strategydev.DefaultLabDetailPATimeout,
		TATimeout:     strategydev.DefaultLabDetailTATimeout,
		JEVTimeout:    strategydev.DefaultLabDetailJEVTimeout,
		CachedMTF: &flow.LabKlineMTF, SessionID: sessionID, RunID: flow.RunID, Jev: jev,
		LongOnly: flow.TradeDirection.LongOnlyEpisodeEval(),
		KlineJEVProfile: strings.TrimSpace(body.KlineJEVProfile),
	}, ref)
	if err != nil {
		if strategydev.IsBadLabRef(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	workflowchat.StoreLabEpisodeDetail(chat, nil, flow.RunID, cacheRef, detail)
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
