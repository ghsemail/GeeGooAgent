package signalclient

import (
	"context"
	"strings"
	"time"
)

// IndexTrendRequest is POST /v1/lab/index-trend (sync).
type IndexTrendRequest struct {
	Code            string     `json:"code"`
	Frequency       string     `json:"frequency"`
	MonthsBack      int        `json:"months_back"`
	AsOf            *time.Time `json:"as_of,omitempty"`
	StrategyVersion string     `json:"strategy_version"`
}

// IndexTrendTrendRow is one catalog trend flag at the last bar.
type IndexTrendTrendRow struct {
	Name   string            `json:"name"`
	Index  string            `json:"index"`
	Type   string            `json:"type"`
	Param  map[string]string `json:"param,omitempty"`
	Stance string            `json:"stance"`
	Value  int               `json:"value"`
}

// IndexTrendSummary aggregates stance counts for one timeframe.
type IndexTrendSummary struct {
	Long  int `json:"long"`
	Short int `json:"short"`
	Flat  int `json:"flat"`
	Net   int `json:"net"`
}

// IndexTrendResponse is the successful lab-api body for one timeframe.
type IndexTrendResponse struct {
	Code            string              `json:"code"`
	Frequency       string              `json:"frequency"`
	StrategyVersion string              `json:"strategy_version"`
	MonthsBack      int                 `json:"months_back"`
	BarCount        int                 `json:"bar_count"`
	LastBarTime     string              `json:"last_bar_time,omitempty"`
	Trends          []IndexTrendTrendRow `json:"trends"`
	Summary         IndexTrendSummary   `json:"summary"`
}

// FetchIndexTrend calls POST /v1/lab/index-trend.
func (c *HTTPClient) FetchIndexTrend(ctx context.Context, req IndexTrendRequest) (IndexTrendResponse, error) {
	req.Code = strings.TrimSpace(req.Code)
	req.Frequency = strings.TrimSpace(req.Frequency)
	req.StrategyVersion = NormalizeStrategyVersion(req.StrategyVersion)
	if req.StrategyVersion == "" {
		req.StrategyVersion = "alpha"
	}
	if req.Code == "" || req.Frequency == "" || req.MonthsBack <= 0 {
		return IndexTrendResponse{}, &APIError{StatusCode: 400, Message: "code, frequency, months_back required"}
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
	var out IndexTrendResponse
	if err := c.postJSON(ctx, "/v1/lab/index-trend", body, &out); err != nil {
		return IndexTrendResponse{}, err
	}
	out.StrategyVersion = NormalizeStrategyVersion(out.StrategyVersion)
	return out, nil
}
