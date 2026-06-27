package memory

// BotStock is one monitored bot from get_report_bot_codes.
type BotStock struct {
	Code      string `json:"code"`
	StockName string `json:"stock_name"`
	BotID     string `json:"bot_id"`
	BotName   string `json:"bot_name"`
	BotType   string `json:"bot_type"`
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
	StockNewsSummary           string `json:"stock_news_summary,omitempty"`
}

// PreMarketWorking is workflow working memory.
type PreMarketWorking struct {
	SessionID      string                    `json:"session_id"`
	Skill          string                    `json:"skill"`
	Phase          string                    `json:"phase"`
	IsTradingDay   *bool                     `json:"is_trading_day"`
	BotCodes       []BotStock                `json:"bot_codes"`
	MarketContext  MarketContext             `json:"market_context"`
	Stocks         map[string]StockWorkspace `json:"stocks"`
	Artifacts      map[string]string         `json:"artifacts"`
	StepsCompleted []string                  `json:"steps_completed"`
	CurrentStock   string                    `json:"current_stock_code"`
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
		Stocks:         map[string]StockWorkspace{},
		Artifacts:      map[string]string{},
		StepsCompleted: []string{},
	}
}
