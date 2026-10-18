package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

func (h *Handler) registerLabRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/lab/experiments", h.labListExperiments)
	mux.HandleFunc("POST /v1/lab/experiments/price-analysis", h.labRunPriceAnalysis)
}

func (h *Handler) labEnabled() bool {
	if h.App == nil || h.App.Config == nil {
		return false
	}
	return h.App.Config.LabEnabled()
}

func (h *Handler) labService() *lab.Service {
	cfg := h.App.Config
	client := signalclient.NewHTTPClient(cfg.LabAPIURL(), cfg.LabAPIKey())
	return lab.NewService(client)
}

func (h *Handler) labGate(w http.ResponseWriter, r *http.Request) bool {
	if !h.labEnabled() {
		writeError(w, http.StatusNotFound, "lab routes disabled")
		return false
	}
	if !requireMCPTokenForChat(w, r, "", h.configMCPToken()) {
		return false
	}
	return true
}

func (h *Handler) labListExperiments(w http.ResponseWriter, r *http.Request) {
	if !h.labGate(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"experiments": []map[string]string{
			{"id": lab.ExperimentPriceAnalysis, "name": "多周期价格分析", "status": "beta"},
		},
	})
}

type labPriceAnalysisBody struct {
	Code            string  `json:"code"`
	SignalFrequency string  `json:"signal_frequency"`
	Months          int     `json:"months"`
	AnchorTime      string  `json:"anchor_time"`
	SpotPrice       float64 `json:"spot_price"`
}

func (h *Handler) labRunPriceAnalysis(w http.ResponseWriter, r *http.Request) {
	if !h.labGate(w, r) {
		return
	}
	var body labPriceAnalysisBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	var anchor *time.Time
	if s := strings.TrimSpace(body.AnchorTime); s != "" {
		ts, err := parseLabAnchorTime(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		anchor = &ts
	}
	resp, err := h.labService().RunPriceAnalysis(r.Context(), lab.PriceAnalysisRequest{
		Code:            body.Code,
		SignalFrequency: body.SignalFrequency,
		Months:          body.Months,
		AnchorTime:      anchor,
		SpotPrice:       body.SpotPrice,
	})
	if err != nil {
		if lab.IsBadRequest(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func parseLabAnchorTime(s string) (time.Time, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if ts, err := time.ParseInLocation(layout, s, loc); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, errBadRequest("invalid anchor_time")
}

type errBadRequest string

func (e errBadRequest) Error() string { return string(e) }
