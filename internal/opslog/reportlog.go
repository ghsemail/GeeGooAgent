package opslog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// DetailEntry is one stock in a report generation batch.
type DetailEntry struct {
	Code       string  `json:"code"`
	StockName  string  `json:"stock_name"`
	Status     string  `json:"status"`
	ReportID   string  `json:"report_id,omitempty"`
	ChangePct  float64 `json:"change_pct,omitempty"`
	Summary    string  `json:"summary,omitempty"`
	Error      string  `json:"error,omitempty"`
	StartedAt  string  `json:"started_at"`
	FinishedAt string  `json:"finished_at"`
}

// RunRecorder collects one user/market report skill execution for ops dashboard.
type RunRecorder struct {
	RunID     string
	RunDate   string
	Phase     string
	Skill     string
	Market    string
	UserID    string
	SessionID string
	StartedAt time.Time
}

// NewRunRecorder starts an ops log for a stock report skill run.
func NewRunRecorder(skill, market, userID, sessionID, reportDate string) *RunRecorder {
	phase := "premarket"
	if skill == "postmarket_stock" {
		phase = "postmarket"
	}
	rd := strings.TrimSpace(reportDate)
	if rd == "" {
		rd = time.Now().Format("2006-01-02")
	}
	return &RunRecorder{
		RunID:     primitive.NewObjectID().Hex(),
		RunDate:   rd,
		Phase:     phase,
		Skill:     skill,
		Market:    strings.ToUpper(strings.TrimSpace(market)),
		UserID:    strings.TrimSpace(userID),
		SessionID: strings.TrimSpace(sessionID),
		StartedAt: time.Now(),
	}
}

// PersistFromResult posts the run log to GeeGooSignal catalog-api.
func PersistFromResult(
	ctx context.Context,
	cfg *config.AppConfig,
	rec *RunRecorder,
	result workflow.RunResult,
	feishuSent bool,
	feishuSkipReason string,
) {
	if rec == nil || cfg == nil {
		return
	}
	base := strings.TrimRight(cfg.SignalCatalogURL(), "/")
	key := strings.TrimSpace(cfg.SignalCatalogAPIKey())
	if base == "" || key == "" {
		slog.Warn("opslog: catalog api not configured, skip report generation log",
			"run_id", rec.RunID, "skill", rec.Skill)
		return
	}

	details := detailsFromWorking(result.Working)
	reported, skipped, failed := countDetails(details)
	status := batchStatus(reported, skipped, failed, len(details))
	supervisorVerdict := ""
	if result.Supervisor != nil {
		supervisorVerdict = string(result.Supervisor.Verdict)
	}
	finished := time.Now()
	doc := map[string]any{
		"run_id":              rec.RunID,
		"run_date":            rec.RunDate,
		"phase":               rec.Phase,
		"skill":               rec.Skill,
		"market":              rec.Market,
		"user_id":             rec.UserID,
		"session_id":          rec.SessionID,
		"status":              status,
		"started_at":          rec.StartedAt.Format("2006-01-02 15:04:05"),
		"finished_at":         finished.Format("2006-01-02 15:04:05"),
		"total_stocks":        len(details),
		"reports_created":     reported,
		"skipped":             skipped,
		"failed":              failed,
		"feishu_sent":         feishuSent,
		"feishu_skip_reason":  strings.TrimSpace(feishuSkipReason),
		"supervisor_verdict":  supervisorVerdict,
		"workflow_status":     result.Status,
		"workflow_last_error": strings.TrimSpace(result.LastError),
		"details":             details,
	}
	if err := postReportLog(ctx, base, key, doc); err != nil {
		slog.Warn("opslog: report generation log persist failed",
			"run_id", rec.RunID, "skill", rec.Skill, "market", rec.Market, "err", err)
	}
}

func postReportLog(ctx context.Context, base, bearer string, doc map[string]any) error {
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/recordReportGenerationLog", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			return err
		}
		if code, ok := out["code"].(float64); ok && int(code) != 100 {
			msg, _ := out["message"].(string)
			if msg == "" {
				msg = "record failed"
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

func detailsFromWorking(w *memory.PreMarketWorking) []DetailEntry {
	if w == nil || len(w.Stocks) == 0 {
		return nil
	}
	now := detailNow()
	out := make([]DetailEntry, 0, len(w.Stocks))
	for _, ws := range w.Stocks {
		entry := DetailEntry{
			Code:       ws.Code,
			StockName:  ws.StockName,
			Status:     ws.Status,
			ReportID:   ws.ReportID,
			ChangePct:  ws.ChangePct,
			Summary:    truncate(ws.ReportSummary, 240),
			StartedAt:  now,
			FinishedAt: now,
		}
		if ws.Status == "failed" {
			entry.Error = "stock workflow failed"
		}
		out = append(out, entry)
	}
	return out
}

func batchStatus(reported, skipped, failed, total int) string {
	if total == 0 {
		return "failed"
	}
	if failed > 0 && reported == 0 && skipped == 0 {
		return "failed"
	}
	if failed > 0 {
		return "partial_failed"
	}
	if reported == 0 && skipped == 0 {
		return "failed"
	}
	return "success"
}

func countDetails(details []DetailEntry) (reported, skipped, failed int) {
	for _, d := range details {
		switch d.Status {
		case "reported":
			reported++
		case "failed":
			failed++
		default:
			skipped++
		}
	}
	return reported, skipped, failed
}

func detailNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if s == "" || n <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
