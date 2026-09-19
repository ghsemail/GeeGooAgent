package chat

import (
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func canonicalSkill(name string) string {
	if name == legacyGenerateStrategyCognition {
		return SkillGenerateStrategyArchive
	}
	return name
}

const (
	MetaKeyActiveFlow = "active_flow"

	SkillMultiStrategyCompare         = "multi_strategy_compare"
	SkillParamTune                    = "param_tune"
	SkillStrategyDev                  = "strategy_dev"
	SkillSignalDiagnose               = "signal_diagnose"
	SkillGenerateStrategyArchive      = "generate_strategy_archive"
	SkillGenerateStrategyCognition    = SkillGenerateStrategyArchive
	legacyGenerateStrategyCognition   = "generate_strategy_cognition"

	StepGenerateCognition = "generate_cognition"
	StepStrategyDevelop   = "develop"

	PhaseDevPick          = "dev_pick"
	PhaseDevEnsureArchive = "dev_ensure_archive"
	PhaseDevReadCognition = "dev_read_cognition" // legacy persisted phase → dev_ensure_archive

	PhaseCognitionPick        = "cognition_pick"
	PhaseCognitionReadCatalog = "cognition_read_catalog"
	PhaseCognitionWebResearch = "cognition_web_research"
	PhaseCognitionCompose     = "cognition_compose"
	PhaseCognitionSaveKB      = "cognition_save_kb"
	PhaseCognitionVerifyKB    = "cognition_verify_kb"

	PhaseSignalDiagPick    = "diag_pick"
	PhaseReadStrategy      = "read_strategy"
	PhaseRunProbe          = "run_probe"
	PhaseFetchKeyLevels    = "fetch_key_levels"
	PhaseEvaluateAccuracy  = "evaluate_accuracy"
	PhaseBuildDetail       = "build_detail"
	PhaseDiagCompose       = "diag_compose"
	PhaseDiagSaveKB        = "diag_save_kb"

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

	defaultMonthsBack       = 3
	maxStrategies           = 4
	maxStepRetries          = 2
	cognitionParseWait = 180 // seconds — WeKnora publish + embed is async
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
	// strategy workflows (generate cognition / strategy dev)
	WorkflowStep   string         `json:"workflow_step,omitempty"`
	StrategyQuery  string         `json:"strategy_query,omitempty"`
	CatalogType    string         `json:"catalog_type,omitempty"`
	CatalogLabel   string         `json:"catalog_label,omitempty"`
	CatalogRaw     map[string]any `json:"catalog_raw,omitempty"`
	WebNotes       []WebNote      `json:"web_notes,omitempty"`
	KBDraft        string         `json:"kb_draft,omitempty"`
	KnowledgeID    string         `json:"knowledge_id,omitempty"`
	KnowledgeTitle string         `json:"knowledge_title,omitempty"`
	VerifySnippet       string         `json:"verify_snippet,omitempty"`
	DevArchiveGenerated bool           `json:"dev_archive_generated,omitempty"`
	ProbeBuyHits        int            `json:"probe_buy_hits,omitempty"`
	ProbeSellHits       int            `json:"probe_sell_hits,omitempty"`
	ProbeBarCount       int            `json:"probe_bar_count,omitempty"`
	DiagnoseVerdict     string         `json:"diagnose_verdict,omitempty"`
	DiagnoseSummary     string         `json:"diagnose_summary,omitempty"`
	DiagnoseRaw         map[string]any `json:"diagnose_raw,omitempty"`
	ProbeRaw            map[string]any `json:"probe_raw,omitempty"`
	SignalEval          slots.SignalEpisodeEval `json:"signal_eval,omitempty"`
	UseKeyLevelEpisodeStop bool           `json:"use_key_level_episode_stop,omitempty"`
	KeyBreakMode           string         `json:"key_break_mode,omitempty"`
	KeyLevels              KeyLevelSnapshot `json:"key_levels,omitempty"`
	EvalJudgment        string         `json:"eval_judgment,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// KeyLevelSnapshot is a diagnose-time Key Level Engine band (shared across probe window in v1).
type KeyLevelSnapshot struct {
	SupportLow    float64 `json:"support_low,omitempty"`
	SupportHigh   float64 `json:"support_high,omitempty"`
	SupportCenter float64 `json:"support_center,omitempty"`
	ResistLow     float64 `json:"resist_low,omitempty"`
	ResistHigh    float64 `json:"resist_high,omitempty"`
	ResistCenter  float64 `json:"resist_center,omitempty"`
	Summary       string  `json:"summary,omitempty"`
}

// WebNote is one web_search result saved into cognition draft.
type WebNote struct {
	Query   string `json:"query"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url,omitempty"`
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
