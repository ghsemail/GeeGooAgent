package signalclient

import (
	"context"
	"strings"
	"time"
)

// TechnicalAnalysisRequest is POST /v1/lab/technical-analysis (sync).
type TechnicalAnalysisRequest struct {
	Code             string     `json:"code"`
	Frequency        string     `json:"frequency"`
	StrategyVersion  string     `json:"strategy_version"`
	MonthsBack       int        `json:"months_back"`
	AsOf             *time.Time `json:"as_of,omitempty"`
}

// TechnicalAnalysisResponse is the successful lab-api body for one timeframe.
type TechnicalAnalysisResponse struct {
	Code            string                    `json:"code"`
	Frequency       string                    `json:"frequency"`
	StrategyVersion string                    `json:"strategy_version"`
	MonthsBack      int                       `json:"months_back"`
	BarCount        int                       `json:"bar_count"`
	LastBarTime     string                    `json:"last_bar_time,omitempty"`
	Indicators      TechnicalAnalysisIndicators `json:"indicators"`
}

// TechnicalAnalysisIndicators holds Elder-style toolbox fields from Signal.
type TechnicalAnalysisIndicators struct {
	EMA         TechnicalAnalysisEMA         `json:"ema"`
	Envelope    TechnicalAnalysisEnvelope    `json:"envelope"`
	MACD        TechnicalAnalysisMACD        `json:"macd"`
	Stochastic  TechnicalAnalysisStochastic  `json:"stochastic"`
	RSI         TechnicalAnalysisRSI         `json:"rsi"`
	ForceIndex  TechnicalAnalysisForceIndex  `json:"force_index"`
}

type TechnicalAnalysisEMA struct {
	EMA13      float64 `json:"ema_13"`
	EMA26      float64 `json:"ema_26"`
	EMA13Slope float64 `json:"ema_13_slope"`
	EMA26Slope float64 `json:"ema_26_slope"`
	ValueZone  string  `json:"value_zone"`
}

type TechnicalAnalysisEnvelope struct {
	MidEMA22  float64 `json:"mid_ema_22"`
	Upper     float64 `json:"upper"`
	Lower     float64 `json:"lower"`
	HalfWidth float64 `json:"half_width"`
}

type TechnicalAnalysisMACD struct {
	Line           float64 `json:"line"`
	Signal         float64 `json:"signal"`
	Histogram      float64 `json:"histogram"`
	HistogramSlope float64 `json:"histogram_slope"`
}

type TechnicalAnalysisStochastic struct {
	SlowK float64 `json:"slow_k"`
	SlowD float64 `json:"slow_d"`
	Zone  string  `json:"zone"`
}

type TechnicalAnalysisRSI struct {
	Value float64 `json:"value"`
	Zone  string  `json:"zone"`
}

type TechnicalAnalysisForceIndex struct {
	EMA2  float64 `json:"ema_2"`
	EMA13 float64 `json:"ema_13"`
}

// NormalizeStrategyVersion maps UI aliases to English lab-api tokens (never send Greek α).
func NormalizeStrategyVersion(raw string) string {
	s := strings.TrimSpace(raw)
	switch strings.ToLower(s) {
	case "", "alpha", "a":
		return "alpha"
	}
	if s == "α" {
		return "alpha"
	}
	return s
}

// FetchTechnicalAnalysis calls POST /v1/lab/technical-analysis.
func (c *HTTPClient) FetchTechnicalAnalysis(ctx context.Context, req TechnicalAnalysisRequest) (TechnicalAnalysisResponse, error) {
	req.Code = strings.TrimSpace(req.Code)
	req.Frequency = strings.TrimSpace(req.Frequency)
	req.StrategyVersion = NormalizeStrategyVersion(req.StrategyVersion)
	if req.StrategyVersion == "" {
		req.StrategyVersion = "alpha"
	}
	if req.Code == "" || req.Frequency == "" || req.MonthsBack <= 0 {
		return TechnicalAnalysisResponse{}, &APIError{StatusCode: 400, Message: "code, frequency, months_back required"}
	}
	body := map[string]any{
		"code":              req.Code,
		"frequency":         req.Frequency,
		"strategy_version":  req.StrategyVersion,
		"months_back":       req.MonthsBack,
	}
	if req.AsOf != nil && !req.AsOf.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		body["as_of"] = req.AsOf.In(loc).Format(asOfLayout)
	}
	var out TechnicalAnalysisResponse
	if err := c.postJSON(ctx, "/v1/lab/technical-analysis", body, &out); err != nil {
		return TechnicalAnalysisResponse{}, err
	}
	out.StrategyVersion = NormalizeStrategyVersion(out.StrategyVersion)
	return out, nil
}
