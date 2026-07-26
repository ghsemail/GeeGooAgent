package runtimeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

type dataProbeRequest struct {
	Checks []string `json:"checks"`
	Market string   `json:"market"`
	Code   string   `json:"code"`
	Limit  int      `json:"limit"`
	MCPToken string `json:"mcp_token,omitempty"`
}

type dataProbeResult struct {
	Check       string   `json:"check"`
	OK          bool     `json:"ok"`
	Path        string   `json:"path"`
	LatencyMS   int64    `json:"latency_ms"`
	ItemCount   int      `json:"item_count,omitempty"`
	SourcesUsed []string `json:"sources_used,omitempty"`
	Detail      string   `json:"detail"`
}

func (h *Handler) dataProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req dataProbeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	checks := normalizeProbeChecks(req.Checks)
	if len(checks) == 0 {
		writeError(w, http.StatusBadRequest, "checks required")
		return
	}
	market := strings.ToUpper(strings.TrimSpace(req.Market))
	if market == "" {
		market = "CN"
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "600519.SH"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	cfg := h.appConfig()
	token := resolveChatMCPToken(r, req.MCPToken, cfg.MCPToken())
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing mcp_token: set X-MCP-Token or agent config mcp_token")
		return
	}

	client := mcp.NewClient(cfg.EffectiveMCPURL(), cfg.MCPAPIKey(), mcp.Options{
		Timeout:      25 * time.Second,
		MaxRetries:   1,
		AllowedHosts: cfg.ResolvedAllowedHosts(),
	})

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	results := make([]dataProbeResult, 0, len(checks))
	allOK := true
	for _, check := range checks {
		res := runDataProbeCheck(ctx, client, token, check, market, code, limit)
		if !res.OK {
			allOK = false
		}
		results = append(results, res)
	}

	writeJSON(w, map[string]any{
		"ok":      allOK,
		"results": results,
	})
}

func normalizeProbeChecks(in []string) []string {
	if len(in) == 0 {
		return []string{"market_news", "stock_news", "quote"}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		check := strings.ToLower(strings.TrimSpace(raw))
		if check == "" {
			continue
		}
		if _, ok := seen[check]; ok {
			continue
		}
		seen[check] = struct{}{}
		out = append(out, check)
	}
	return out
}

func runDataProbeCheck(
	ctx context.Context,
	client *mcp.Client,
	token, check, market, code string,
	limit int,
) dataProbeResult {
	start := time.Now()
	switch check {
	case "market_news":
		data, err := client.GetMarketNews(ctx, token, market, limit)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			return dataProbeResult{
				Check: "market_news", OK: false, Path: "bot", LatencyMS: latency,
				Detail: err.Error(),
			}
		}
		text := strings.TrimSpace(data.Text)
		count := len(data.Items)
		if count == 0 && text != "" {
			count = 1
		}
		ok := text != "" && !strings.Contains(text, "暂无数据")
		detail := fmt.Sprintf("%s market news: %d items", market, count)
		if !ok {
			detail = fmt.Sprintf("%s market news: API OK but empty", market)
		}
		return dataProbeResult{
			Check: "market_news", OK: ok, Path: "bot", LatencyMS: latency,
			ItemCount: count, SourcesUsed: data.SourcesUsed, Detail: detail,
		}
	case "stock_news":
		data, err := client.GetStockNews(ctx, token, code, limit)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			return dataProbeResult{
				Check: "stock_news", OK: false, Path: "bot", LatencyMS: latency,
				Detail: err.Error(),
			}
		}
		text := strings.TrimSpace(data.Text)
		count := len(data.Items)
		if count == 0 && text != "" {
			count = 1
		}
		ok := text != "" && !strings.Contains(text, "暂无数据")
		detail := fmt.Sprintf("%s stock news: %d items", code, count)
		if !ok {
			detail = fmt.Sprintf("%s stock news: API OK but empty", code)
		}
		return dataProbeResult{
			Check: "stock_news", OK: ok, Path: "bot", LatencyMS: latency,
			ItemCount: count, SourcesUsed: data.SourcesUsed, Detail: detail,
		}
	case "quote":
		data, err := client.GetCurrentPrice(ctx, token, code)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			return dataProbeResult{
				Check: "quote", OK: false, Path: "bot", LatencyMS: latency,
				Detail: err.Error(),
			}
		}
		ok := data != nil && data.Price > 0
		detail := fmt.Sprintf("%s quote unavailable", code)
		if ok {
			detail = fmt.Sprintf("%s price %.4f", code, data.Price)
		}
		return dataProbeResult{
			Check: "quote", OK: ok, Path: "bot", LatencyMS: latency,
			ItemCount: boolToCount(ok), Detail: detail,
		}
	case "capital_flow":
		flows, err := client.GetCapitalFlow(ctx, token, code, "DAY", "")
		latency := time.Since(start).Milliseconds()
		if err != nil {
			return dataProbeResult{
				Check: "capital_flow", OK: false, Path: "bot", LatencyMS: latency,
				Detail: err.Error(),
			}
		}
		ok := len(flows) > 0
		detail := fmt.Sprintf("%s capital flow: %d point(s)", code, len(flows))
		if !ok {
			detail = fmt.Sprintf("%s capital flow: API OK but empty", code)
		}
		return dataProbeResult{
			Check: "capital_flow", OK: ok, Path: "bot", LatencyMS: latency,
			ItemCount: len(flows), Detail: detail,
		}
	default:
		return dataProbeResult{
			Check: check, OK: false, Path: "bot",
			Detail: "unknown check: " + check,
		}
	}
}

func boolToCount(ok bool) int {
	if ok {
		return 1
	}
	return 0
}
