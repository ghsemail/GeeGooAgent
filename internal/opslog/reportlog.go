package opslog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

const (
	reportLogCollection = "report_generation_log"
	defaultOpsMongoDB   = "Signal_DB"
)

// DetailEntry is one stock in a report generation batch.
type DetailEntry struct {
	Code       string  `bson:"code" json:"code"`
	StockName  string  `bson:"stock_name" json:"stock_name"`
	Status     string  `bson:"status" json:"status"`
	ReportID   string  `bson:"report_id,omitempty" json:"report_id,omitempty"`
	ChangePct  float64 `bson:"change_pct,omitempty" json:"change_pct,omitempty"`
	Summary    string  `bson:"summary,omitempty" json:"summary,omitempty"`
	Error      string  `bson:"error,omitempty" json:"error,omitempty"`
	StartedAt  string  `bson:"started_at" json:"started_at"`
	FinishedAt string  `bson:"finished_at" json:"finished_at"`
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

// PersistFromResult writes the run log to Mongo (Signal_DB.report_generation_log).
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
	if err := insertReportLog(ctx, cfg, doc); err != nil {
		slog.Warn("opslog: report generation log persist failed",
			"run_id", rec.RunID, "skill", rec.Skill, "market", rec.Market, "err", err)
	}
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

var (
	mongoOnce sync.Once
	mongoCli  *mongo.Client
	mongoErr  error
)

func insertReportLog(ctx context.Context, cfg *config.AppConfig, doc map[string]any) error {
	coll, err := reportLogColl(ctx, cfg)
	if err != nil {
		return err
	}
	_, err = coll.InsertOne(ctx, doc)
	return err
}

func reportLogColl(ctx context.Context, cfg *config.AppConfig) (*mongo.Collection, error) {
	uri, dbName := opsMongo(cfg)
	if uri == "" {
		return nil, fmt.Errorf("ops mongo not configured")
	}
	mongoOnce.Do(func() {
		mongoCli, mongoErr = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	})
	if mongoErr != nil {
		return nil, mongoErr
	}
	if dbName == "" {
		dbName = defaultOpsMongoDB
	}
	return mongoCli.Database(dbName).Collection(reportLogCollection), nil
}

func opsMongo(cfg *config.AppConfig) (uri, db string) {
	if cfg == nil {
		return "", ""
	}
	uri = strings.TrimSpace(cfg.OpsMongoURI)
	db = strings.TrimSpace(cfg.OpsMongoDB)
	if db == "" {
		db = defaultOpsMongoDB
	}
	return uri, db
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
