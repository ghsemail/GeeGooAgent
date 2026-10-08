package strategydev

import "github.com/ghsemail/GeeGooAgent/internal/slots"

// Input is the six-step strategy development orchestration request.
type Input struct {
	Code        string
	StockName   string
	Frequency   string
	MonthsBack  int // analysis window, capped at 6 months
	MonthsBackH int // optional longer window hint for step 6 (Jev only)
}

// Result is the workflow output after all applicable steps.
type Result struct {
	Code       string `json:"code"`
	Frequency  string `json:"frequency"`
	BarLimit   int    `json:"bar_limit"`
	MonthsBack int    `json:"months_back"`

	Regime     string  `json:"regime"`
	Side       string  `json:"side"`
	Confidence float64 `json:"confidence,omitempty"`
	RegimeNote string  `json:"regime_note,omitempty"`

	Skipped bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`

	BuyCandidates  []Candidate `json:"buy_candidates,omitempty"`
	SellCandidates []Candidate `json:"sell_candidates,omitempty"`

	SelectedBuy  *Candidate `json:"selected_buy,omitempty"`
	SelectedSell *Candidate `json:"selected_sell,omitempty"`
	BuyFallbackNote string  `json:"buy_fallback_note,omitempty"`

	Diagnose DiagnoseSummary `json:"diagnose"`

	FinalVerdict string `json:"final_verdict"` // 通过 | 换一对 | 拉长窗口
	VerdictReason string `json:"verdict_reason,omitempty"`
	JevVerdict    string `json:"jev_verdict,omitempty"`
}

// Candidate is one catalog strategy eligible for Jev selection.
type Candidate struct {
	SignalID string `json:"signal_id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // index | combination | advanced
	Buy      []any  `json:"buy_signal,omitempty"`
	Sell     []any  `json:"sell_signal,omitempty"`
}

// DiagnoseSummary holds probe/diagnose episode metrics for step 5–6.
type DiagnoseSummary struct {
	BuyHits      int     `json:"buy_hits"`
	SellHits     int     `json:"sell_hits"`
	CompleteSegs int     `json:"complete_segments"`
	HitRate      float64 `json:"hit_rate"`
	AvgDirection float64 `json:"avg_direction_return"`
	CumPoints    float64 `json:"cumulative_points_pct"`
	PrimarySide  string  `json:"primary_side"` // buy | sell
	Eval         slots.SignalEpisodeEval `json:"-"`
}

const (
	RegimeUp        = "上涨"
	RegimeDown      = "下跌"
	RegimeRange     = "横盘"
	RegimeRevert    = "回归"
	RegimeUnclear   = "不明确"
	RegimeWeakUp    = "弱上涨"
	RegimeWeakDown  = "弱下跌"

	SideLeft  = "左侧"
	SideRight = "右侧"
	SideNone  = "不判"

	VerdictPass      = "通过"
	VerdictSwapPair  = "换一对"
	VerdictLongWindow = "拉长窗口"

	coverComboID = "662d0424c4cee7ffb800d0af"
	specialRevertComboID = "667d2683"
)
