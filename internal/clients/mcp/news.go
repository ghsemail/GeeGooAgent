package mcp

import (
	"context"
	"encoding/json"
)

// GetMarketNews calls POST /getMarketNews.
func (c *Client) GetMarketNews(ctx context.Context, mcpToken, market string, limit int) (*MarketNewsData, error) {
	body := map[string]any{
		"mcp_token": mcpToken,
		"market":    market,
	}
	if limit > 0 {
		body["limit"] = limit
	}
	payload, err := c.Post(ctx, "/getMarketNews", body)
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return nil, newClientError("missing data in getMarketNews response", nil, 0)
	}
	b, _ := json.Marshal(dataRaw)
	var out MarketNewsData
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetStockNews calls POST /getStockNews.
func (c *Client) GetStockNews(ctx context.Context, mcpToken, code string, limit int) (*StockNewsData, error) {
	body := map[string]any{
		"mcp_token": mcpToken,
		"code":      code,
	}
	if limit > 0 {
		body["limit"] = limit
	}
	payload, err := c.Post(ctx, "/getStockNews", body)
	if err != nil {
		return nil, err
	}
	dataRaw, ok := payload["data"].(map[string]any)
	if !ok {
		return nil, newClientError("missing data in getStockNews response", nil, 0)
	}
	b, _ := json.Marshal(dataRaw)
	var out StockNewsData
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
