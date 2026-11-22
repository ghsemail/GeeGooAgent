package diagnoseskill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 120 * time.Second

// LaneClient fetches locked macro-api :3220 diagnose fact lanes.
type LaneClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// LaneBundle holds raw lane JSON from macro-api (pass-through arrays in skill result).
type LaneBundle struct {
	PriceAnalysis      json.RawMessage
	KlinePattern       json.RawMessage
	TechnicalAnalysis  json.RawMessage
	IndexTrend         json.RawMessage
	SpotPrice          float64
	SignalFrequency    string
	Code               string
}

func (c *LaneClient) httpClient() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func (c *LaneClient) post(ctx context.Context, path string, body any) (json.RawMessage, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		return nil, fmt.Errorf("macro-api base url not configured")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(c.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(c.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("macro-api %s: %s", path, msg)
	}
	var envelope json.RawMessage
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, err
	}
	return envelope, nil
}

// FetchStrategyLanes loads four strategy diagnose lanes for one symbol.
func (c *LaneClient) FetchStrategyLanes(ctx context.Context, code, signalFreq, asOf string, months int) (LaneBundle, error) {
	code = strings.TrimSpace(code)
	signalFreq = strings.TrimSpace(signalFreq)
	if code == "" || signalFreq == "" {
		return LaneBundle{}, fmt.Errorf("code and signal_frequency required")
	}
	base := map[string]any{
		"code":             code,
		"signal_frequency": signalFreq,
		"months_back":      months,
	}
	if asOf != "" {
		base["as_of"] = asOf
	}
	nosignal := []any{map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}}}
	klineBody := cloneMap(base)
	klineBody["buy_signal"] = nosignal
	klineBody["sell_signal"] = nosignal

	var out LaneBundle
	out.Code = code
	out.SignalFrequency = signalFreq
	var err error
	if out.KlinePattern, err = c.post(ctx, "/v1/diagnose/strategy/kline-pattern", klineBody); err != nil {
		return LaneBundle{}, err
	}
	if out.PriceAnalysis, err = c.post(ctx, "/v1/diagnose/strategy/price-analysis", base); err != nil {
		return LaneBundle{}, err
	}
	if out.TechnicalAnalysis, err = c.post(ctx, "/v1/diagnose/strategy/technical-analysis", base); err != nil {
		return LaneBundle{}, err
	}
	if out.IndexTrend, err = c.post(ctx, "/v1/diagnose/strategy/index-trend", base); err != nil {
		return LaneBundle{}, err
	}
	out.SpotPrice = spotPriceFromLane(out.PriceAnalysis)
	return out, nil
}

// FetchMacroLanes loads four macro diagnose lanes (daily only).
func (c *LaneClient) FetchMacroLanes(ctx context.Context, code, asOf string, months int) (LaneBundle, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return LaneBundle{}, fmt.Errorf("code required")
	}
	base := map[string]any{"code": code, "months_back": months}
	if asOf != "" {
		base["as_of"] = asOf
	}
	var out LaneBundle
	out.Code = code
	var err error
	if out.KlinePattern, err = c.post(ctx, "/v1/diagnose/macro/kline-pattern", base); err != nil {
		return LaneBundle{}, err
	}
	if out.PriceAnalysis, err = c.post(ctx, "/v1/diagnose/macro/price-analysis", base); err != nil {
		return LaneBundle{}, err
	}
	if out.TechnicalAnalysis, err = c.post(ctx, "/v1/diagnose/macro/technical-analysis", base); err != nil {
		return LaneBundle{}, err
	}
	if out.IndexTrend, err = c.post(ctx, "/v1/diagnose/macro/index-trend", base); err != nil {
		return LaneBundle{}, err
	}
	out.SpotPrice = spotPriceFromLane(out.PriceAnalysis)
	return out, nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func spotPriceFromLane(raw json.RawMessage) float64 {
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return 0
	}
	if v, ok := doc["spot_price"].(float64); ok {
		return v
	}
	return 0
}

func laneArray(raw json.RawMessage, key string) ([]map[string]any, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	arr, ok := doc[key].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, it := range arr {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}