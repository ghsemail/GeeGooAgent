package mcp

import (
	"context"
	"encoding/json"
)

// CheckTradingDay calls POST /checkTradingDay.
func (c *Client) CheckTradingDay(ctx context.Context, mcpToken, code string) (*TradingDayData, error) {
	payload, err := c.Post(ctx, "/checkTradingDay", map[string]any{
		"mcp_token": mcpToken,
		"code":      code,
	})
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return nil, newClientError("missing data in checkTradingDay response", nil, 0)
	}
	return decodeTradingDay(dataRaw)
}

// GetReportBotCodes calls POST /getReportBotCodes.
func (c *Client) GetReportBotCodes(ctx context.Context, mcpToken string) ([]UserBotCode, error) {
	payload, err := c.Post(ctx, "/getReportBotCodes", map[string]any{"mcp_token": mcpToken})
	if err != nil {
		return nil, err
	}
	items, ok := payload["data"].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]UserBotCode, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		b, _ := json.Marshal(m)
		var bot UserBotCode
		_ = json.Unmarshal(b, &bot)
		out = append(out, bot)
	}
	return out, nil
}

// GetMCPAnalysis calls POST /getMCPAnalysis.
func (c *Client) GetMCPAnalysis(ctx context.Context, mcpToken string, name, code, promptID, period, language string) (*McpAnalysisData, error) {
	if language == "" {
		language = "cn"
	}
	payload, err := c.Post(ctx, "/getMCPAnalysis", map[string]any{
		"mcp_token": mcpToken,
		"name":      name,
		"code":      code,
		"prompt_id": promptID,
		"period":    period,
		"language":  language,
	})
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return &McpAnalysisData{}, nil
	}
	b, _ := json.Marshal(dataRaw)
	var result McpAnalysisData
	_ = json.Unmarshal(b, &result)
	return &result, nil
}

// GetCapitalFlow calls POST /getCapitalFlow.
func (c *Client) GetCapitalFlow(ctx context.Context, mcpToken, code, period, start string) ([]CapitalFlowItem, error) {
	body := map[string]any{
		"mcp_token": mcpToken,
		"code":      code,
		"period":    period,
	}
	if start != "" {
		body["start"] = start
	}
	payload, err := c.Post(ctx, "/getCapitalFlow", body)
	if err != nil {
		return nil, err
	}
	items, ok := payload["data"].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]CapitalFlowItem, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		b, _ := json.Marshal(m)
		var flow CapitalFlowItem
		_ = json.Unmarshal(b, &flow)
		out = append(out, flow)
	}
	return out, nil
}

// GetCapitalDistribution calls POST /getCapitalDistribution.
func (c *Client) GetCapitalDistribution(ctx context.Context, mcpToken, code string) (*CapitalDistributionData, error) {
	payload, err := c.Post(ctx, "/getCapitalDistribution", map[string]any{
		"mcp_token": mcpToken,
		"code":      code,
	})
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return &CapitalDistributionData{}, nil
	}
	b, _ := json.Marshal(dataRaw)
	var dist CapitalDistributionData
	_ = json.Unmarshal(b, &dist)
	return &dist, nil
}

// GetBotYesterdayAttitude calls POST /getBotYesterdayAttitude.
func (c *Client) GetBotYesterdayAttitude(ctx context.Context, mcpToken, botID, language string) (*BotYesterdayAttitude, error) {
	if language == "" {
		language = "cn"
	}
	payload, err := c.Post(ctx, "/getBotYesterdayAttitude", map[string]any{
		"mcp_token": mcpToken,
		"bot_id":    botID,
		"language":  language,
	})
	if err != nil {
		if ce, ok := err.(*ClientError); ok {
			if ce.APICode != nil && *ce.APICode == 105 {
				return neutralAttitude(botID), nil
			}
			if ce.HTTPStatus == 404 {
				return neutralAttitude(botID), nil
			}
		}
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return neutralAttitude(botID), nil
	}
	b, _ := json.Marshal(dataRaw)
	var attitude BotYesterdayAttitude
	_ = json.Unmarshal(b, &attitude)
	attitude.Found = true
	return &attitude, nil
}

// CreatePreMarketReport calls POST /createPreMarketReport.
func (c *Client) CreatePreMarketReport(ctx context.Context, mcpToken string, body map[string]any) (*PreMarketReportResult, error) {
	req := map[string]any{"mcp_token": mcpToken}
	for k, v := range body {
		req[k] = v
	}
	payload, err := c.Post(ctx, "/createPreMarketReport", req)
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return &PreMarketReportResult{}, nil
	}
	reportID, _ := dataRaw["report_id"].(string)
	return &PreMarketReportResult{ReportID: reportID}, nil
}

// GetStockDailyReports calls POST /getStockDailyReports.
func (c *Client) GetStockDailyReports(ctx context.Context, mcpToken, code, reportDate string) (*DailyReportsData, error) {
	payload, err := c.Post(ctx, "/getStockDailyReports", map[string]any{
		"mcp_token":   mcpToken,
		"code":        code,
		"report_date": reportDate,
	})
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return &DailyReportsData{}, nil
	}
	b, _ := json.Marshal(dataRaw)
	var reports DailyReportsData
	_ = json.Unmarshal(b, &reports)
	return &reports, nil
}

// SearchCode calls POST /searchCode (no mcp_token).
func (c *Client) SearchCode(ctx context.Context, regex string, markets []string) ([]SearchCodeItem, error) {
	body := map[string]any{"regex": regex}
	if len(markets) > 0 {
		marketAny := make([]any, len(markets))
		for i, m := range markets {
			marketAny[i] = m
		}
		body["market"] = marketAny
	}
	raw, err := c.PostDirect(ctx, "/searchCode", body)
	if err != nil {
		return nil, err
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, nil
	}
	out := make([]SearchCodeItem, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, parseSearchCodeItem(m))
	}
	return out, nil
}

// GetCurrentPrice calls POST /getCurrentPrice.
func (c *Client) GetCurrentPrice(ctx context.Context, mcpToken, code string) (*CurrentPriceData, error) {
	raw, err := c.PostDirect(ctx, "/getCurrentPrice", map[string]any{
		"mcp_token": mcpToken,
		"code":      code,
	})
	if err != nil {
		return nil, err
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, newClientError("unexpected getCurrentPrice response", nil, 0)
	}
	price, _ := m["price"].(float64)
	return &CurrentPriceData{Price: price}, nil
}

func decodeTradingDay(data map[string]any) (*TradingDayData, error) {
	b, _ := json.Marshal(data)
	var td TradingDayData
	if err := json.Unmarshal(b, &td); err != nil {
		return nil, err
	}
	return &td, nil
}

func neutralAttitude(botID string) *BotYesterdayAttitude {
	return &BotYesterdayAttitude{
		Attitude: "neutral",
		BotID:    botID,
		Found:    false,
	}
}
