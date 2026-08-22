package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const (
	dataBFFNodeTimeout    = 8 * time.Second
	dataBFFOverviewTTL    = 30 * time.Second
	dataBFFOverviewBudget = 30 * time.Second
)

type dataFleetCache struct {
	mu        sync.Mutex
	expiresAt time.Time
	payload   map[string]any
}

var globalDataFleetCache dataFleetCache

type dataNodeHealth struct {
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type dataNodeFutu struct {
	OK   bool   `json:"ok"`
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
}

type dataNodeExchange struct {
	OK     bool   `json:"ok"`
	Source string `json:"source,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type dataNodeCapabilities struct {
	Regions     []string `json:"regions,omitempty"`
	Quote       bool     `json:"quote"`
	CapitalFlow bool     `json:"capital_flow"`
	News        bool     `json:"news"`
}

type dataNodeNewsSummary struct {
	EnabledSources  int `json:"enabled_sources"`
	HealthySources  int `json:"healthy_sources"`
	CacheMarketTTL  int `json:"cache_market_ttl_sec,omitempty"`
	CacheStockTTL   int `json:"cache_stock_ttl_sec,omitempty"`
}

type dataFleetNode struct {
	ID           string               `json:"id"`
	Label        string               `json:"label"`
	BaseURL      string               `json:"base_url"`
	Regions      []string             `json:"regions,omitempty"`
	Health       dataNodeHealth       `json:"health"`
	Futu         *dataNodeFutu        `json:"futu,omitempty"`
	Exchange     *dataNodeExchange    `json:"exchange,omitempty"`
	Capabilities dataNodeCapabilities `json:"capabilities"`
	News         dataNodeNewsSummary  `json:"news"`
}

type dataFleetCollector struct {
	client *http.Client
}

func newDataFleetCollector() *dataFleetCollector {
	return &dataFleetCollector{
		client: &http.Client{Timeout: dataBFFNodeTimeout},
	}
}

func (h *Handler) collectDataOverview(ctx context.Context, force bool) (map[string]any, error) {
	if !force {
		if cached := globalDataFleetCache.get(); cached != nil {
			return cached, nil
		}
	}
	cfg := h.appConfig()
	nodes := cfg.ResolvedDataNodes()
	collector := newDataFleetCollector()

	outNodes := make([]dataFleetNode, len(nodes))
	botURL := cfg.EffectiveMCPURL()
	var botOK bool
	var botDetail string

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node config.ResolvedDataNode) {
			defer wg.Done()
			outNodes[i] = collector.probeNode(ctx, node)
		}(i, node)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		botOK, botDetail = collector.probeBotHealth(ctx, botURL)
	}()
	wg.Wait()

	summary := map[string]int{
		"nodes_total":     len(outNodes),
		"nodes_healthy":   0,
		"sources_total":   0,
		"sources_healthy": 0,
	}
	allOK := true
	nodePayloads := make([]map[string]any, 0, len(outNodes))
	for _, n := range outNodes {
		if n.Health.OK {
			summary["nodes_healthy"]++
		} else {
			allOK = false
		}
		summary["sources_total"] += n.News.EnabledSources
		summary["sources_healthy"] += n.News.HealthySources
		raw, _ := json.Marshal(n)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		nodePayloads = append(nodePayloads, m)
	}
	if !botOK {
		allOK = false
	}

	payload := map[string]any{
		"ok":         allOK,
		"checked_at": time.Now().UTC().Format(time.RFC3339),
		"nodes":      nodePayloads,
		"routing": map[string]any{
			"bot_mcp_url": botURL,
			"bot_ok":      botOK,
			"bot_detail":  botDetail,
		},
		"summary": summary,
	}
	globalDataFleetCache.set(payload, dataBFFOverviewTTL)
	return payload, nil
}

func (c *dataFleetCollector) probeNode(ctx context.Context, node config.ResolvedDataNode) dataFleetNode {
	out := dataFleetNode{
		ID:      node.ID,
		Label:   node.Label,
		BaseURL: node.BaseURL,
		Regions: append([]string(nil), node.Regions...),
		Capabilities: dataNodeCapabilities{
			Quote:       true,
			CapitalFlow: true,
			News:        true,
		},
	}
	out.Health = c.getHealth(ctx, node)

	if caps, ok := c.getCapabilities(ctx, node); ok {
		if len(out.Regions) == 0 {
			out.Regions = caps.Regions
		}
		out.Capabilities.Regions = caps.Regions
	}
	if sources, ok := c.getNewsSources(ctx, node); ok {
		out.News.EnabledSources = countEnabledNewsSources(sources)
		out.News.CacheMarketTTL = intFromAny(sources["cache_market_ttl_sec"])
		out.News.CacheStockTTL = intFromAny(sources["cache_stock_ttl_sec"])
		if health, ok := c.getNewsHealth(ctx, node); ok {
			out.News.HealthySources = countHealthyEnabledNewsSources(sources, health)
		}
	} else if health, ok := c.getNewsHealth(ctx, node); ok {
		out.News.HealthySources = countHealthyNewsSources(health)
	}
	if nodeServesFutu(out.Regions) {
		if futu, ok := c.getFutuHealth(ctx, node); ok {
			out.Futu = &futu
		}
	}
	if nodeServesCrypto(out.Regions) {
		if exchange, ok := c.getCryptoHealth(ctx, node); ok {
			out.Exchange = &exchange
		}
	}
	return out
}

func (c *dataFleetCollector) getHealth(ctx context.Context, node config.ResolvedDataNode) dataNodeHealth {
	start := time.Now()
	body, status, err := c.doGET(ctx, node.BaseURL+"/health", "")
	if err != nil {
		return dataNodeHealth{OK: false, Detail: err.Error()}
	}
	ok := status >= 200 && status < 300
	detail := fmt.Sprintf("HTTP %d %s", status, truncateText(body, 120))
	return dataNodeHealth{OK: ok, LatencyMS: time.Since(start).Milliseconds(), Detail: detail}
}

func (c *dataFleetCollector) getCapabilities(ctx context.Context, node config.ResolvedDataNode) (dataNodeCapabilities, bool) {
	body, status, err := c.doGET(ctx, node.BaseURL+"/v1/market/capabilities", node.Bearer)
	if err != nil || status < 200 || status >= 300 {
		return dataNodeCapabilities{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return dataNodeCapabilities{}, false
	}
	regions := stringListFromAny(raw["regions"])
	return dataNodeCapabilities{
		Regions:     regions,
		Quote:       true,
		CapitalFlow: true,
		News:        true,
	}, true
}

func (c *dataFleetCollector) getNewsSources(ctx context.Context, node config.ResolvedDataNode) (map[string]any, bool) {
	body, status, err := c.doGET(ctx, node.BaseURL+"/v1/news/sources", node.Bearer)
	if err != nil || status < 200 || status >= 300 {
		return nil, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false
	}
	return raw, true
}

func (c *dataFleetCollector) getNewsHealth(ctx context.Context, node config.ResolvedDataNode) (map[string]any, bool) {
	body, status, err := c.doGET(ctx, node.BaseURL+"/v1/news/health", node.Bearer)
	if err != nil || status < 200 || status >= 300 {
		return nil, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false
	}
	return raw, true
}

func (c *dataFleetCollector) getFutuHealth(ctx context.Context, node config.ResolvedDataNode) (dataNodeFutu, bool) {
	body, status, err := c.doGET(ctx, node.BaseURL+"/v1/futu/health", node.Bearer)
	if err != nil || status < 200 || status >= 300 {
		return dataNodeFutu{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return dataNodeFutu{OK: true}, true
	}
	return dataNodeFutu{
		OK:   futuHealthOK(raw),
		Host: stringFromAny(raw["host"]),
		Port: intFromAny(raw["port"]),
	}, true
}

func futuHealthOK(raw map[string]any) bool {
	if raw["ok"] == true {
		return true
	}
	// GeeGooData returns code:100 on success (see GET /v1/futu/health).
	if intFromAny(raw["code"]) == 100 {
		return true
	}
	return strings.EqualFold(stringFromAny(raw["message"]), "ok")
}

func (c *dataFleetCollector) getCryptoHealth(ctx context.Context, node config.ResolvedDataNode) (dataNodeExchange, bool) {
	body, status, err := c.doGET(ctx, node.BaseURL+"/v1/crypto/health", node.Bearer)
	if err != nil || status < 200 || status >= 300 {
		return dataNodeExchange{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return dataNodeExchange{OK: true, Source: "binance"}, true
	}
	return dataNodeExchange{
		OK:     raw["ok"] == true || intFromAny(raw["code"]) == 100,
		Source: stringFromAny(raw["source"]),
		Detail: stringFromAny(raw["detail"]),
	}, true
}

func nodeServesFutu(regions []string) bool {
	for _, r := range regions {
		switch strings.ToUpper(strings.TrimSpace(r)) {
		case "CN", "HK", "US", "SH", "SZ":
			return true
		}
	}
	return false
}

func nodeServesCrypto(regions []string) bool {
	for _, r := range regions {
		if strings.EqualFold(strings.TrimSpace(r), "CRYPTO") {
			return true
		}
	}
	return false
}

func (c *dataFleetCollector) probeBotHealth(ctx context.Context, botURL string) (bool, string) {
	body, status, err := c.doGET(ctx, strings.TrimSuffix(botURL, "/")+"/health", "")
	if err != nil {
		return false, err.Error()
	}
	ok := status >= 200 && status < 300
	return ok, fmt.Sprintf("HTTP %d %s", status, truncateText(body, 80))
}

func (c *dataFleetCollector) proxyGET(ctx context.Context, node config.ResolvedDataNode, path string) (map[string]any, int, error) {
	return c.proxy(ctx, node, http.MethodGet, path, nil)
}

func (c *dataFleetCollector) proxy(ctx context.Context, node config.ResolvedDataNode, method, path string, body []byte) (map[string]any, int, error) {
	raw, status, err := c.doRequest(ctx, strings.TrimSuffix(node.BaseURL, "/")+path, node.Bearer, method, body)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	if status < 200 || status >= 300 {
		return nil, status, fmt.Errorf("upstream HTTP %d: %s", status, truncateText(raw, 200))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, http.StatusBadGateway, err
	}
	out["node_id"] = node.ID
	return out, http.StatusOK, nil
}

func (c *dataFleetCollector) doGET(ctx context.Context, url, bearer string) ([]byte, int, error) {
	return c.doRequest(ctx, url, bearer, http.MethodGet, nil)
}

func (c *dataFleetCollector) doRequest(ctx context.Context, url, bearer, method string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func (h *Handler) appConfig() *config.AppConfig {
	if h.App != nil && h.App.Config != nil {
		return h.App.Config
	}
	return &config.AppConfig{}
}

func (c *dataFleetCache) get() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.payload == nil || time.Now().After(c.expiresAt) {
		return nil
	}
	out := make(map[string]any, len(c.payload))
	for k, v := range c.payload {
		out[k] = v
	}
	return out
}

func (c *dataFleetCache) set(payload map[string]any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.payload = payload
	c.expiresAt = time.Now().Add(ttl)
}

func (c *dataFleetCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.payload = nil
	c.expiresAt = time.Time{}
}

func countEnabledNewsSources(sources map[string]any) int {
	regions, ok := sources["regions"].(map[string]any)
	if !ok {
		return 0
	}
	total := 0
	for _, rv := range regions {
		region, ok := rv.(map[string]any)
		if !ok {
			continue
		}
		total += countEnabledSourceList(region["market_sources"])
		total += countEnabledSourceList(region["stock_sources"])
		// legacy flat list
		total += countEnabledSourceList(region["sources"])
	}
	return total
}

func countEnabledSourceList(v any) int {
	list, ok := v.([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["enabled"] == false {
			continue
		}
		n++
	}
	return n
}

func countHealthyNewsSources(health map[string]any) int {
	list, ok := health["sources"].([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["ok"] == true {
			n++
		}
	}
	return n
}

// countHealthyEnabledNewsSources counts enabled config slots whose source_id probe is ok.
// Matches countEnabledNewsSources so overview ratios stay 1:1 when every probe passes.
func countHealthyEnabledNewsSources(sources, health map[string]any) int {
	probeOK := probeOKBySourceID(health)
	return countEnabledNewsSourcesMatching(sources, func(id string) bool {
		ok, known := probeOK[id]
		return known && ok
	})
}

func probeOKBySourceID(health map[string]any) map[string]bool {
	out := make(map[string]bool)
	list, ok := health["sources"].([]any)
	if !ok {
		return out
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := stringFromAny(m["id"])
		if id == "" {
			continue
		}
		out[id] = m["ok"] == true
	}
	return out
}

func countEnabledNewsSourcesMatching(sources map[string]any, match func(id string) bool) int {
	regions, ok := sources["regions"].(map[string]any)
	if !ok {
		return 0
	}
	total := 0
	for _, rv := range regions {
		region, ok := rv.(map[string]any)
		if !ok {
			continue
		}
		total += countMatchingSourceList(region["market_sources"], match)
		total += countMatchingSourceList(region["stock_sources"], match)
		total += countMatchingSourceList(region["sources"], match)
	}
	return total
}

func countMatchingSourceList(v any, match func(id string) bool) int {
	list, ok := v.([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["enabled"] == false {
			continue
		}
		id := stringFromAny(m["id"])
		if id != "" && match(id) {
			n++
		}
	}
	return n
}

func stringListFromAny(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func truncateText(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
