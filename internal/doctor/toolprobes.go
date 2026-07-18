package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/tools/newsrunner"
)

const probeCodeHK = "00700.HK"

// checkToolProbes smoke-tests fragile L2 tools after base connectivity passes.
// API/auth errors are FAIL; empty payloads are WARN (non-trading / optional data).
func checkToolProbes(cfg *config.AppConfig) []CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	client := mcp.NewClient(cfg.EffectiveMCPURL(), cfg.MCPAPIKey(), mcp.Options{
		Timeout:      25 * time.Second,
		MaxRetries:   1,
		AllowedHosts: cfg.ResolvedAllowedHosts(),
	})
	token := cfg.MCPToken()

	var results []CheckResult
	results = append(results, probeSearchCode(ctx, cfg))
	results = append(results, probeCapitalFlow(ctx, client, token))
	results = append(results, probeCapitalDistribution(ctx, client, token))
	results = append(results, probeMCPCodeQuery(ctx, client, token, "get_position", "/getPosition"))
	results = append(results, probeMCPCodeQuery(ctx, client, token, "get_ticker", "/getTicker"))
	results = append(results, probeMCPCodeQuery(ctx, client, token, "get_broker", "/getBroker"))
	results = append(results, probeStockNews(ctx, client, token))
	results = append(results, probeMarketNews(ctx, client, token))
	results = append(results, probeStockDailyReports(ctx, client, token))
	results = append(results, CheckResult{
		Name:   "tool probe: get_mcp_analysis",
		OK:     true,
		Detail: "skipped (LLM 60–180s); analyze /health 已测",
	})
	return results
}

func probeSearchCode(ctx context.Context, cfg *config.AppConfig) CheckResult {
	name := "tool probe: search_code"
	client := mcp.NewClient(cfg.SignalAPIURL(), cfg.SignalAPIKey(), mcp.Options{
		Timeout:      25 * time.Second,
		AllowedHosts: cfg.ResolvedAllowedHosts(),
	})
	items, err := client.SearchCode(ctx, "00700", nil)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	if len(items) == 0 {
		return CheckResult{Name: name, OK: true, Warn: true, Detail: "API OK but 0 matches for 00700"}
	}
	return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("%d match(es) for 00700", len(items))}
}

func probeCapitalFlow(ctx context.Context, client *mcp.Client, token string) CheckResult {
	name := "tool probe: get_capital_flow"
	for _, period := range []string{"DAY", "WEEK"} {
		flows, err := client.GetCapitalFlow(ctx, token, probeCodeHK, period, "")
		if err != nil {
			return CheckResult{Name: name, OK: false, Detail: err.Error()}
		}
		if len(flows) > 0 {
			return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("%s %s: %d point(s)", probeCodeHK, period, len(flows))}
		}
	}
	return CheckResult{
		Name: name, OK: true, Warn: true,
		Detail: probeCodeHK + ": API OK but empty (non-trading or Bot→Data CN route)",
	}
}

func probeCapitalDistribution(ctx context.Context, client *mcp.Client, token string) CheckResult {
	name := "tool probe: get_capital_distribution"
	dist, err := client.GetCapitalDistribution(ctx, token, probeCodeHK)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	if probeCapitalDistHasData(dist) {
		return CheckResult{Name: name, OK: true, Detail: probeCodeHK + ": distribution present"}
	}
	return CheckResult{
		Name: name, OK: true, Warn: true,
		Detail: probeCodeHK + ": API OK but empty distribution",
	}
}

func probeCapitalDistHasData(d *mcp.CapitalDistributionData) bool {
	if d == nil {
		return false
	}
	fields := []float64{
		d.CapitalInSuper, d.CapitalInBig, d.CapitalInMid, d.CapitalInSmall,
		d.CapitalOutSuper, d.CapitalOutBig, d.CapitalOutMid, d.CapitalOutSmall,
	}
	for _, v := range fields {
		if v != 0 {
			return true
		}
	}
	return strings.TrimSpace(d.UpdateTime) != ""
}

func probeMCPCodeQuery(ctx context.Context, client *mcp.Client, token, toolName, path string) CheckResult {
	name := "tool probe: " + toolName
	body := map[string]any{
		"mcp_token": token,
		"code":      probeCodeHK,
	}
	if toolName == "get_ticker" {
		body["num"] = 3
	}
	payload, err := client.Post(ctx, path, body)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	data := payload["data"]
	if probePayloadEmpty(data) {
		return CheckResult{
			Name: name, OK: true, Warn: true,
			Detail: probeCodeHK + ": API OK but empty (OpenD off-hours or no Futu account)",
		}
	}
	return CheckResult{Name: name, OK: true, Detail: probePayloadSummary(data)}
}

func probePayloadEmpty(data any) bool {
	switch v := data.(type) {
	case nil:
		return true
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

func probePayloadSummary(data any) string {
	switch v := data.(type) {
	case []any:
		return fmt.Sprintf("%d item(s)", len(v))
	case map[string]any:
		return fmt.Sprintf("%d field(s)", len(v))
	default:
		return "ok"
	}
}

func probeStockNews(ctx context.Context, client *mcp.Client, token string) CheckResult {
	name := "tool probe: fetch_stock_news"
	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	data, err := client.GetStockNews(probeCtx, token, probeCodeHK, 3)
	if err != nil {
		// Fall back to local Go probe when Bot route not deployed yet.
		text, localErr := newsrunner.StockNewsGo(probeCtx, probeCodeHK, 3)
		if localErr != nil {
			return CheckResult{Name: name, OK: false, Detail: err.Error()}
		}
		text = strings.TrimSpace(text)
		if text == "" || strings.Contains(text, "暂无数据") {
			return CheckResult{
				Name: name, OK: true, Warn: true,
				Detail: probeCodeHK + ": Bot news unavailable; local probe empty",
			}
		}
		return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("%s: local fallback %d chars", probeCodeHK, len(text))}
	}
	text := strings.TrimSpace(data.Text)
	if text == "" || strings.Contains(text, "暂无数据") {
		return CheckResult{
			Name: name, OK: true, Warn: true,
			Detail: probeCodeHK + ": API OK but no headlines",
		}
	}
	src := strings.Join(data.SourcesUsed, ",")
	if src == "" {
		src = "GeeGooData-via-Bot"
	}
	return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("%s via %s: %d chars", probeCodeHK, src, len(text))}
}

func probeMarketNews(ctx context.Context, client *mcp.Client, token string) CheckResult {
	name := "tool probe: fetch_market_news"
	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	data, err := client.GetMarketNews(probeCtx, token, "HK", 3)
	if err != nil {
		return CheckResult{Name: name, OK: true, Warn: true, Detail: "HK market news via Bot: " + err.Error()}
	}
	text := strings.TrimSpace(data.Text)
	if text == "" || strings.Contains(text, "暂无数据") {
		return CheckResult{Name: name, OK: true, Warn: true, Detail: "HK market news: API OK but empty"}
	}
	src := strings.Join(data.SourcesUsed, ",")
	if src == "" {
		src = "GeeGooData-via-Bot"
	}
	return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("HK via %s: %d chars", src, len(text))}
}

func probeStockDailyReports(ctx context.Context, client *mcp.Client, token string) CheckResult {
	name := "tool probe: get_stock_daily_reports"
	reportDate := time.Now().Format("2006-01-02")
	reports, err := client.GetStockDailyReports(ctx, token, probeCodeHK, reportDate)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	n := len(reports.PreMarket) + len(reports.PostMarket)
	if n == 0 {
		return CheckResult{
			Name: name, OK: true, Warn: true,
			Detail: fmt.Sprintf("%s %s: no reports (normal if not generated today)", probeCodeHK, reportDate),
		}
	}
	return CheckResult{Name: name, OK: true, Detail: fmt.Sprintf("%s %s: %d report(s)", probeCodeHK, reportDate, n)}
}
