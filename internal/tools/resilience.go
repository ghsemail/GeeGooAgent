package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/search"
	"github.com/ghsemail/GeeGooAgent/internal/tools/newsrunner"
)

const mcpRetryPause = 2 * time.Second

// httpEmptyRetryTools are catalog HTTP tools that retry once when the API
// returns code=100 but an empty payload (transient OpenD / analyze-api gaps).
var httpEmptyRetryTools = map[string]bool{
	"get_position":           true,
	"get_ticker":             true,
	"get_broker":             true,
	"generate_grid_strategy": true,
	"generate_dca_strategy":  true,
}

func shouldRetryHTTPEmpty(toolName string) bool {
	return httpEmptyRetryTools[toolName]
}

func waitRetry(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(mcpRetryPause):
		return true
	}
}

func capitalDistributionHasData(d *mcp.CapitalDistributionData) bool {
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

func fetchCapitalFlowResilient(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, code, period string,
) ([]mcp.CapitalFlowItem, string, error) {
	periods := []string{period}
	if period == "" || period == "DAY" {
		periods = []string{"DAY", "WEEK"}
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		for _, p := range periods {
			flows, err := client.GetCapitalFlow(ctx, mcpToken, code, p, "")
			if err != nil {
				lastErr = err
				continue
			}
			if len(flows) > 0 {
				return flows, p, nil
			}
		}
		if attempt == 0 {
			select {
			case <-ctx.Done():
				return nil, period, ctx.Err()
			case <-time.After(mcpRetryPause):
			}
		}
	}
	if lastErr != nil {
		return nil, period, lastErr
	}
	return nil, period, nil
}

func fetchCapitalDistributionResilient(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, code string,
) (*mcp.CapitalDistributionData, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		dist, err := client.GetCapitalDistribution(ctx, mcpToken, code)
		if err != nil {
			lastErr = err
		} else if capitalDistributionHasData(dist) {
			return dist, nil
		}
		if attempt == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(mcpRetryPause):
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return &mcp.CapitalDistributionData{}, nil
}

func stockNewsNeedsFallback(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	return strings.Contains(text, "暂无数据") || strings.Contains(text, "no items")
}

func newsItemsToAny(items []mcp.NewsItem) []any {
	if len(items) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"title":        item.Title,
			"url":          item.URL,
			"snippet":      item.Snippet,
			"source_id":    item.SourceID,
			"source_label": item.SourceLabel,
			"published_at": item.PublishedAt,
		})
	}
	return out
}

func fetchMarketNewsResilient(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, projectRoot, market string,
	limit int,
) (text, source string, items []any, err error) {
	if client != nil && strings.TrimSpace(mcpToken) != "" {
		var lastErr error
		for attempt := 0; attempt < 2; attempt++ {
			data, mcpErr := client.GetMarketNews(ctx, mcpToken, market, limit)
			if mcpErr == nil && data != nil && !stockNewsNeedsFallback(data.Text) {
				return data.Text, "GeeGooData-via-Bot", newsItemsToAny(data.Items), nil
			}
			if mcpErr != nil {
				lastErr = mcpErr
			}
			if attempt == 0 && waitRetry(ctx) {
				continue
			}
			if lastErr != nil {
				err = lastErr
			}
			break
		}
	}
	text, localErr := newsrunner.MarketNews(ctx, newsrunner.Options{ProjectRoot: projectRoot}, market, limit)
	if localErr == nil && !stockNewsNeedsFallback(text) {
		return text, "finance-news-local", []any{}, nil
	}
	if errors.Is(localErr, newsrunner.ErrUnavailable) {
		return "", "", nil, newsrunner.ErrUnavailable
	}
	if localErr != nil && err == nil {
		err = localErr
	}
	return text, "", nil, err
}

func fetchStockNewsResilient(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, projectRoot, code string,
	limit int,
) (text, source string, items []any, err error) {
	if client != nil && strings.TrimSpace(mcpToken) != "" {
		var lastErr error
		for attempt := 0; attempt < 2; attempt++ {
			data, mcpErr := client.GetStockNews(ctx, mcpToken, code, limit)
			if mcpErr == nil && data != nil && !stockNewsNeedsFallback(data.Text) {
				return data.Text, "GeeGooData-via-Bot", newsItemsToAny(data.Items), nil
			}
			if mcpErr != nil {
				lastErr = mcpErr
			}
			if attempt == 0 && waitRetry(ctx) {
				continue
			}
			if lastErr != nil {
				err = lastErr
			}
			break
		}
	}
	text, localErr := newsrunner.StockNews(ctx, newsrunner.Options{ProjectRoot: projectRoot}, code, limit)
	if localErr == nil && !stockNewsNeedsFallback(text) {
		return text, "finance-news-local", []any{}, nil
	}
	if errors.Is(localErr, newsrunner.ErrUnavailable) {
		return "", "", nil, newsrunner.ErrUnavailable
	}
	if localErr != nil && err == nil {
		err = localErr
	}
	return text, "", nil, err
}

func getMCPAnalysisResilient(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, name, code, promptID, period, language string,
) (*mcp.McpAnalysisData, error) {
	var lastErr error
	var last *mcp.McpAnalysisData
	for attempt := 0; attempt < 2; attempt++ {
		result, err := client.GetMCPAnalysis(ctx, mcpToken, name, code, promptID, period, language)
		if err != nil {
			lastErr = err
		} else {
			last = result
			if result != nil && strings.TrimSpace(result.AnalysisResult) != "" {
				return result, nil
			}
		}
		if attempt == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(mcpRetryPause):
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	if last == nil {
		last = &mcp.McpAnalysisData{}
	}
	return last, nil
}

func recallFromMCPReports(
	ctx context.Context,
	client *mcp.Client,
	mcpToken, code, startDate string,
	maxDays int,
) (summary, source string, found bool) {
	if maxDays <= 0 {
		maxDays = 3
	}
	for d := 0; d < maxDays; d++ {
		reportDate := shiftDate(startDate, -d)
		reports, err := client.GetStockDailyReports(ctx, mcpToken, code, reportDate)
		if err != nil {
			continue
		}
		if text := firstReportField(reports.PreMarket); text != "" {
			return truncateRecall(text, 4000), fmt.Sprintf("mcp:pre_market:%s", reportDate), true
		}
		if text := firstReportField(reports.PostMarket); text != "" {
			return truncateRecall(text, 4000), fmt.Sprintf("mcp:post_market:%s", reportDate), true
		}
	}
	return "", "", false
}

func firstReportField(items []map[string]any) string {
	for _, m := range items {
		for _, key := range []string{"content", "report_content", "summary", "text"} {
			if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func shiftDate(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func webSearchNewsFallback(ctx context.Context, cfg config.SearchConfig, code string) (string, []map[string]any) {
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = search.ProviderDuckDuckGo
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 8
	}
	query := stockNewsQuery(code)
	hits, err := search.Search(ctx, search.Config{
		Provider: cfg.Provider, MaxResults: cfg.MaxResults,
	}, query)
	if err != nil || len(hits) == 0 {
		return "", nil
	}
	items := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		items = append(items, map[string]any{
			"title": h.Title, "url": h.URL, "snippet": h.Snippet,
		})
	}
	return formatWebSearchNews(code, query, items), items
}

func webSearchMarketFallback(ctx context.Context, cfg config.SearchConfig, market string) (string, []map[string]any) {
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = search.ProviderDuckDuckGo
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 8
	}
	query := marketNewsQuery(market)
	hits, err := search.Search(ctx, search.Config{
		Provider: cfg.Provider, MaxResults: cfg.MaxResults,
	}, query)
	if err != nil || len(hits) == 0 {
		return "", nil
	}
	items := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		items = append(items, map[string]any{
			"title": h.Title, "url": h.URL, "snippet": h.Snippet,
		})
	}
	return formatMarketWebSearchNews(market, query, items), items
}

func marketNewsQuery(market string) string {
	market = strings.ToUpper(strings.TrimSpace(market))
	switch market {
	case "CN", "HK":
		return market + " 股市 新闻"
	default:
		return market + " stock market news today"
	}
}

func buildMarketNewsResult(market, text, source string, items []any) Result {
	if stockNewsNeedsFallback(text) {
		return newsUnavailableResult("fetch_market_news", market, "",
			fmt.Errorf("no headlines after GeeGooData and web_search"))
	}
	summary := fmt.Sprintf("fetch_market_news %s: %d chars", market, len(text))
	if text == "" {
		summary = fmt.Sprintf("fetch_market_news %s: no items", market)
	}
	if items == nil {
		items = []any{}
	}
	return Result{Status: StatusOK, Summary: summary, Data: map[string]any{
		"market": market, "text": text, "items": items, "source": source,
	}}
}

func stockNewsQuery(code string) string {
	base := strings.TrimSuffix(strings.TrimSuffix(code, ".HK"), ".US")
	base = strings.TrimSuffix(strings.TrimSuffix(base, ".SH"), ".SZ")
	base = strings.TrimLeft(base, "0")
	if isAShare(code) || strings.HasSuffix(code, ".HK") {
		return base + " 股票 新闻"
	}
	return base + " stock news"
}

func mergeStockNewsText(primary, supplement string) string {
	supplement = strings.TrimSpace(supplement)
	if supplement == "" {
		return primary
	}
	if strings.TrimSpace(primary) == "" || strings.Contains(primary, "暂无数据") {
		return supplement
	}
	return primary + "\n" + supplement
}

func formatWebSearchNews(code, query string, hits []map[string]any) string {
	var b strings.Builder
	b.WriteString("\n## 【个股新闻：" + code + "（web_search 补充）】\n\n")
	for i, h := range hits {
		title, _ := h["title"].(string)
		snippet, _ := h["snippet"].(string)
		url, _ := h["url"].(string)
		if strings.TrimSpace(title) == "" {
			continue
		}
		b.WriteString("**")
		b.WriteString(strings.TrimSpace(title))
		b.WriteString("**\n")
		if snippet != "" {
			b.WriteString("   " + strings.TrimSpace(snippet) + "\n")
		}
		if url != "" {
			b.WriteString("   🔗 " + url + "\n")
		}
		b.WriteString("\n")
		if i >= 7 {
			break
		}
	}
	return b.String()
}

func formatMarketWebSearchNews(market, query string, hits []map[string]any) string {
	var b strings.Builder
	b.WriteString("\n## 【市场新闻：" + market + "（web_search 补充）】\n\n")
	for i, h := range hits {
		title, _ := h["title"].(string)
		snippet, _ := h["snippet"].(string)
		url, _ := h["url"].(string)
		if strings.TrimSpace(title) == "" {
			continue
		}
		b.WriteString("**")
		b.WriteString(strings.TrimSpace(title))
		b.WriteString("**\n")
		if snippet != "" {
			b.WriteString("   " + strings.TrimSpace(snippet) + "\n")
		}
		if url != "" {
			b.WriteString("   🔗 " + url + "\n")
		}
		b.WriteString("\n")
		if i >= 7 {
			break
		}
	}
	_ = query
	return b.String()
}
