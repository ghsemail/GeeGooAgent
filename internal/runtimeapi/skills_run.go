package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (h *Handler) registerSkillsRunRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/skills/run", h.skillsRun)
}

type skillsRunRequest struct {
	Skill    string            `json:"skill"`
	MCPToken string            `json:"mcp_token,omitempty"`
	Intraday *intradayRunInput `json:"intraday,omitempty"`
}

type intradayRunInput struct {
	Code           string `json:"code"`
	StockName      string `json:"stock_name"`
	BotID          string `json:"bot_id"`
	BotName        string `json:"bot_name"`
	BotType        string `json:"bot_type"`
	Frequency      string `json:"frequency"`
	TradeType      string `json:"trade_type"`
	ReportDate     string `json:"report_date,omitempty"`
	AttitudeSwitch *bool  `json:"attitude_switch,omitempty"`
}

type skillsRunResponse struct {
	Status    string         `json:"status"`
	SessionID string         `json:"session_id,omitempty"`
	Skill     string         `json:"skill"`
	Decision  map[string]any `json:"decision,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func (h *Handler) skillsRun(w http.ResponseWriter, r *http.Request) {
	if h.App == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "app not ready"})
		return
	}
	var req skillsRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	skill := normalizeSkillName(strings.TrimSpace(req.Skill))
	if skill == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "skill is required"})
		return
	}

	var runOpts app.SkillRunOptions
	runOpts.MCPToken = strings.TrimSpace(req.MCPToken)
	if skill == "intraday_stock" {
		if req.Intraday == nil || strings.TrimSpace(req.Intraday.Code) == "" {
			writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "intraday.code is required"})
			return
		}
		runOpts.Intraday = &workflow.IntradayInput{
			Code:           strings.TrimSpace(req.Intraday.Code),
			StockName:      strings.TrimSpace(req.Intraday.StockName),
			BotID:          strings.TrimSpace(req.Intraday.BotID),
			BotName:        strings.TrimSpace(req.Intraday.BotName),
			BotType:        strings.TrimSpace(req.Intraday.BotType),
			Frequency:      strings.TrimSpace(req.Intraday.Frequency),
			TradeType:      strings.TrimSpace(req.Intraday.TradeType),
			ReportDate:     strings.TrimSpace(req.Intraday.ReportDate),
			AttitudeSwitch: req.Intraday.AttitudeSwitch,
		}
	}

	result, err := h.App.RunSkillContext(r.Context(), skill, runOpts)
	if err != nil {
		writeJSONStatus(w, http.StatusBadRequest, skillsRunResponse{
			Status: "failed",
			Skill:  skill,
			Error:  err.Error(),
		})
		return
	}

	resp := skillsRunResponse{
		Status:    result.Status,
		SessionID: result.SessionID,
		Skill:     skill,
	}
	if skill == "intraday_stock" && result.Working != nil && req.Intraday != nil {
		resultStr, confidence, reportID := intradayDecisionFromWorking(result.Working, req.Intraday.Code)
		resp.Decision = map[string]any{
			"result":     resultStr,
			"confidence": confidence,
			"report_id":  reportID,
		}
	}
	if result.LastError != "" {
		resp.Error = result.LastError
	}
	writeJSONStatus(w, http.StatusOK, resp)
}

// normalizeSkillName maps external skill aliases to registry names.
func normalizeSkillName(skill string) string {
	switch skill {
	case "intraday":
		return "intraday_stock"
	case "pre_market":
		return "premarket_stock"
	case "post_market":
		return "postmarket_stock"
	default:
		return skill
	}
}

func intradayDecisionFromWorking(w *memory.PreMarketWorking, code string) (result, confidence, reportID string) {
	result, confidence, reportID = "hold", "low", ""
	if w == nil {
		return
	}
	code = strings.TrimSpace(code)
	ws, ok := w.Stocks[code]
	if !ok {
		for k, v := range w.Stocks {
			code, ws, ok = k, v, true
			break
		}
	}
	if !ok {
		return
	}
	result, confidence = ws.IntradayResult, ws.IntradayConfidence
	if result == "" {
		result, confidence = workflow.DecideIntraday(ws)
	}
	reportID = ws.ReportID
	return
}
