package chat

import (
	"time"
)

const (
	MetaKeyActiveFlow = "active_flow"

	SkillMultiStrategyCompare = "multi_strategy_compare"
	SkillParamTune            = "param_tune"

	PhaseResolveSymbol  = "resolve_symbol"
	PhasePickStrategies = "pick_strategies"
	PhaseProbeForeach   = "probe_foreach"
	PhaseSummarize      = "summarize"
	PhaseDone           = "done"

	StatusRunning      = "running"
	StatusPausedFailed = "paused_failed"
	StatusCompleted    = "completed"
	StatusInterrupted  = "interrupted"
	StatusCancelled    = "cancelled"

	StepPending = "pending"
	StepRunning = "running"
	StepDone    = "done"
	StepSkipped = "skipped"
	StepFailed  = "failed"

	defaultMonthsBack = 3
	maxStrategies     = 4
	maxStepRetries    = 2
)

// Flow is a persisted serial workflow executed across one or more chat turns.
type Flow struct {
	RunID         string         `json:"run_id"`
	Template      string         `json:"template"`
	Status        string         `json:"status"`
	Phase         string         `json:"phase"`
	Cursor        int            `json:"cursor"`
	StockQuery    string         `json:"stock_query,omitempty"`
	StockCode     string         `json:"stock_code,omitempty"`
	StockName     string         `json:"stock_name,omitempty"`
	MonthsBack    int            `json:"months_back,omitempty"`
	Strategies    []StrategyItem `json:"strategies,omitempty"`
	PartialReport string         `json:"partial_report,omitempty"`
	TriggerText   string         `json:"trigger_text,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// StrategyItem is one strategy slot inside multi_strategy_compare.
type StrategyItem struct {
	Query      string `json:"query"`
	Kind       string `json:"kind,omitempty"`
	Label      string `json:"label,omitempty"`
	Frequency  string `json:"frequency,omitempty"`
	MonthsBack int    `json:"months_back,omitempty"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts,omitempty"`
	LastError  string `json:"last_error,omitempty"`
	BuyHits    int    `json:"buy_hits,omitempty"`
	SellHits   int    `json:"sell_hits,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

// Active reports whether the flow should intercept the next chat turn.
func (f *Flow) Active() bool {
	if f == nil {
		return false
	}
	switch f.Status {
	case StatusRunning, StatusPausedFailed, StatusInterrupted:
		return true
	default:
		return false
	}
}

func (f *Flow) touch() {
	if f == nil {
		return
	}
	f.UpdatedAt = time.Now().UTC()
}
