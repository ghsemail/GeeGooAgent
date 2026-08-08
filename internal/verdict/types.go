package verdict

// Verdict is the finalized trading stance after rule arbitration.
type Verdict struct {
	Result     string
	Confidence string
	Note       string // optional audit note appended to reason
}

// StockPreMarketInput feeds the stock pre-market arbitrator (scheme B).
type StockPreMarketInput struct {
	Attitude               string
	EvidenceCount          int
	HasWeekly              bool
	HasCapitalFlow         bool
	HasCapitalDistribution bool
	HasStockNews           bool
	CapitalRequired        bool // false for A-shares where capital APIs are skipped

	SuggestedResult     string
	SuggestedConfidence string

	MarketResult     string
	MarketConfidence string
}

// MarketPreMarketInput feeds the market-level pre-market arbitrator.
type MarketPreMarketInput struct {
	IndicesDone    bool
	MarketNewsDone bool
	EvidenceCount  int

	SuggestedResult     string
	SuggestedConfidence string
}
