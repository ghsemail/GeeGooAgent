package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// BotStock is one monitored bot from get_report_bot_codes.
type BotStock struct {
	Code      string `json:"code"`
	StockName string `json:"stock_name"`
	BotID     string `json:"bot_id"`
	BotName   string `json:"bot_name"`
	BotType   string `json:"bot_type"`
}

// EvidenceRef is a stable, auditable reference to an observed tool payload.
type EvidenceRef struct {
	ID          string `json:"id"`
	RunID       string `json:"run_id"`
	Tool        string `json:"tool"`
	Source      string `json:"source"`
	ObservedAt  string `json:"observed_at"`
	PayloadHash string `json:"payload_hash"`
	Summary     string `json:"summary"`
}

// MarketContext holds phase A global data.
type MarketContext struct {
	IndicesDone       bool              `json:"indices_done"`
	MarketNewsDone    bool              `json:"market_news_done"`
	IndexAnalysisRefs map[string]string `json:"index_analysis_refs"`
	IndexCodesDone    []string          `json:"index_codes_done"`
	MarketNews        map[string]string `json:"market_news"`
}

// StockWorkspace is per-stock working state in phase B.
type StockWorkspace struct {
	Code                       string `json:"code"`
	StockName                  string `json:"stock_name"`
	BotID                      string `json:"bot_id"`
	BotName                    string `json:"bot_name"`
	BotType                    string `json:"bot_type"`
	Status                     string `json:"status"`
	WeeklyAnalysisRef          string `json:"weekly_analysis_ref,omitempty"`
	Attitude                   string `json:"attitude,omitempty"`
	CapitalFlowSummary         string `json:"capital_flow_summary,omitempty"`
	CapitalDistributionSummary string `json:"capital_distribution_summary,omitempty"`
	ReportRef                  string `json:"report_ref,omitempty"`
	ReportID                   string `json:"report_id,omitempty"`
	StockNewsSummary           string  `json:"stock_news_summary,omitempty"`
	Frequency                  string  `json:"frequency,omitempty"`
	TradeType                  string  `json:"trade_type,omitempty"`
	ReportDate                 string  `json:"report_date,omitempty"`
	PositionSummary            string  `json:"position_summary,omitempty"`
	HasPosition                bool    `json:"has_position,omitempty"`
	PreMarketResult            string  `json:"pre_market_result,omitempty"`
	PreMarketConfidence        string  `json:"pre_market_confidence,omitempty"`
	PreMarketReason            string  `json:"pre_market_reason,omitempty"`
	PreMarketSuggestion        string  `json:"pre_market_suggestion,omitempty"`
	PreMarketReportID          string  `json:"pre_market_report_id,omitempty"`
	HourlyPriceAnalysis        string  `json:"hourly_price_analysis,omitempty"`
	HourlySignalAnalysis       string  `json:"hourly_signal_analysis,omitempty"`
	HourlyKlineAnalysis        string  `json:"hourly_kline_analysis,omitempty"`
	CurrentPrice               float64 `json:"current_price,omitempty"`
	PriceSource                string  `json:"price_source,omitempty"`
	IntradayResult             string  `json:"intraday_result,omitempty"`
	IntradayConfidence         string  `json:"intraday_confidence,omitempty"`
	IntradayReason             string  `json:"intraday_reason,omitempty"`
	BotLogSummary              string  `json:"bot_log_summary,omitempty"`
	ChangePct                  float64 `json:"change_pct,omitempty"`
	SessionBias                string  `json:"session_bias,omitempty"`
	VsPreMarket                string  `json:"vs_pre_market,omitempty"`
}

// PreMarketWorking is workflow working memory.
type PreMarketWorking struct {
	SessionID        string                    `json:"session_id"`
	Skill            string                    `json:"skill"`
	Phase            string                    `json:"phase"`
	IsTradingDay     *bool                     `json:"is_trading_day"`
	BotCodes         []BotStock                `json:"bot_codes"`
	MarketContext    MarketContext             `json:"market_context"`
	Stocks           map[string]StockWorkspace `json:"stocks"`
	Artifacts        map[string]string         `json:"artifacts"`
	EvidenceRefs     []EvidenceRef             `json:"evidence_refs"`
	StepsCompleted   []string                  `json:"steps_completed"`
	CompletedStepKeys []string                 `json:"completed_step_keys,omitempty"`
	CurrentStock     string                    `json:"current_stock_code"`
}

// NewPreMarketWorking creates initial working state.
func NewPreMarketWorking(sessionID, skill string) *PreMarketWorking {
	return &PreMarketWorking{
		SessionID: sessionID,
		Skill:     skill,
		Phase:     "init",
		MarketContext: MarketContext{
			IndexAnalysisRefs: map[string]string{},
			IndexCodesDone:    []string{},
			MarketNews:        map[string]string{},
		},
		Stocks:            map[string]StockWorkspace{},
		Artifacts:         map[string]string{},
		EvidenceRefs:      []EvidenceRef{},
		StepsCompleted:    []string{},
		CompletedStepKeys: []string{},
	}
}

// NewEvidenceRef creates a deterministic evidence ID from the run, source, tool,
// and canonical payload hash. ObservedAt is intentionally excluded from the ID.
func NewEvidenceRef(runID, tool, source, summary string, payload any, observedAt time.Time) EvidenceRef {
	payloadHash := PayloadHash(payload)
	idHash := sha256.Sum256([]byte(strings.Join([]string{runID, tool, source, payloadHash}, "\x00")))
	return EvidenceRef{
		ID:          "ev_" + hex.EncodeToString(idHash[:])[:12],
		RunID:       runID,
		Tool:        tool,
		Source:      source,
		ObservedAt:  observedAt.UTC().Format(time.RFC3339Nano),
		PayloadHash: payloadHash,
		Summary:     OneLine(summary, 240),
	}
}

// PayloadHash returns a canonical SHA-256 hash for evidence payloads.
func PayloadHash(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(strings.TrimSpace(toString(payload)))
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// OneLine normalizes user-facing evidence summaries.
func OneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}
