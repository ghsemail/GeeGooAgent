package mcp

// TradingDayData from /checkTradingDay.
type TradingDayData struct {
	IsTradingDay bool   `json:"is_trading_day"`
	Date         string `json:"date"`
	Market       string `json:"market"`
	Code         string `json:"code"`
}

// UserBotCode from /getReportBotCodes.
type UserBotCode struct {
	StockName string `json:"stock_name"`
	Code      string `json:"code"`
	BotID     string `json:"bot_id"`
	BotName   string `json:"bot_name"`
	BotType   string `json:"bot_type"`
}

// McpAnalysisData from /getMCPAnalysis.
type McpAnalysisData struct {
	AnalysisResult string `json:"analysis_result"`
	Model          string `json:"model,omitempty"`
	CreateDate     string `json:"create_date,omitempty"`
}

// CapitalFlowItem from /getCapitalFlow.
type CapitalFlowItem struct {
	InFlow              float64 `json:"in_flow"`
	MainInFlow          float64 `json:"main_in_flow"`
	SuperInFlow         float64 `json:"super_in_flow"`
	BigInFlow           float64 `json:"big_in_flow"`
	MidInFlow           float64 `json:"mid_in_flow"`
	SmlInFlow           float64 `json:"sml_in_flow"`
	CapitalFlowItemTime string  `json:"capital_flow_item_time,omitempty"`
	LastValidTime       string  `json:"last_valid_time,omitempty"`
}

// CapitalDistributionData from /getCapitalDistribution.
type CapitalDistributionData struct {
	CapitalInSuper  float64 `json:"capital_in_super"`
	CapitalInBig    float64 `json:"capital_in_big"`
	CapitalInMid    float64 `json:"capital_in_mid"`
	CapitalInSmall  float64 `json:"capital_in_small"`
	CapitalOutSuper float64 `json:"capital_out_super"`
	CapitalOutBig   float64 `json:"capital_out_big"`
	CapitalOutMid   float64 `json:"capital_out_mid"`
	CapitalOutSmall float64 `json:"capital_out_small"`
	UpdateTime      string  `json:"update_time,omitempty"`
}

// BotYesterdayAttitude from /getBotYesterdayAttitude.
type BotYesterdayAttitude struct {
	Attitude       string `json:"attitude"`
	AnalysisReport string `json:"analysis_report"`
	BotID          string `json:"bot_id"`
	Code           string `json:"code"`
	StockName      string `json:"stock_name"`
	Date           string `json:"date,omitempty"`
	Language       string `json:"language,omitempty"`
	Found          bool   `json:"-"`
}

// PreMarketReportResult from /createPreMarketReport.
type PreMarketReportResult struct {
	ReportID string `json:"report_id"`
}

// DailyReportsData from /getStockDailyReports.
type DailyReportsData struct {
	PreMarket  []map[string]any `json:"pre_market"`
	Intraday   []map[string]any `json:"intraday"`
	PostMarket []map[string]any `json:"post_market"`
}

// SearchCodeItem from /searchCode.
type SearchCodeItem struct {
	Code      string `json:"code"`
	Name      string `json:"name"` // display: init > en > zh_hk
	NameEN    string `json:"name_en,omitempty"`
	NameZH    string `json:"name_zh,omitempty"`
	Market    string `json:"market,omitempty"`
	StockType string `json:"stock_type,omitempty"`
	LotSize   int    `json:"lot_size,omitempty"`
}

// CurrentPriceData from /getCurrentPrice.
type CurrentPriceData struct {
	Price float64 `json:"price"`
}
