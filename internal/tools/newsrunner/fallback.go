package newsrunner

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const sinaAPI = "https://feed.mix.sina.com.cn/api/roll/get"

var (
	cnbcRSS = "https://www.cnbc.com/id/100003114/device/rss/rss.html"
	djRSS   = "https://feeds.a.dj.com/rss/RSSMarketsMain.xml"
)

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

type rssChannel struct {
	Items []rssItem `xml:"channel>item"`
}

// MarketNewsGo fetches market headlines without Python.
func MarketNewsGo(ctx context.Context, market string, limit int) (string, error) {
	market = strings.ToUpper(strings.TrimSpace(market))
	if limit <= 0 {
		limit = 8
	}
	var parts []string
	switch market {
	case "US":
		items, err := fetchUSIndustryNews(ctx, limit)
		if err != nil {
			return "", err
		}
		parts = append(parts, formatNewsBlock("美股行业动态", items))
	case "CN":
		items, err := fetchSinaRoll(ctx, "", limit*3, nil)
		if err != nil {
			return "", err
		}
		parts = append(parts, formatSinaBlock("A股市场", filterCN(items), limit))
	case "HK":
		items, err := fetchSinaRoll(ctx, "", 50, nil)
		if err != nil {
			return "", err
		}
		parts = append(parts, formatSinaBlock("港股市场", filterHK(items), limit))
	case "ALL":
		if us, err := fetchUSIndustryNews(ctx, limit); err == nil {
			parts = append(parts, formatNewsBlock("美股行业动态", us))
		}
		if items, err := fetchSinaRoll(ctx, "", limit*3, nil); err == nil {
			parts = append(parts, formatSinaBlock("A股市场", filterCN(items), limit))
		}
		if items, err := fetchSinaRoll(ctx, "", 50, nil); err == nil {
			parts = append(parts, formatSinaBlock("港股市场", filterHK(items), limit))
		}
	default:
		return "", fmt.Errorf("unsupported market %s", market)
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if text == "" {
		return "（暂无数据）", nil
	}
	return text, nil
}

// StockNewsGo fetches stock headlines without Python.
func StockNewsGo(ctx context.Context, code string, limit int) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", fmt.Errorf("code required")
	}
	if limit <= 0 {
		limit = 8
	}
	market, ticker := parseStockCode(code)
	lines := []string{fmt.Sprintf("\n## 【个股新闻：%s (%s)】\n", ticker, code)}
	switch market {
	case "US":
		items, err := fetchYahooRSS(ctx, ticker, limit)
		if err != nil {
			return "", err
		}
		lines = append(lines, formatStockItems("美股", items)...)
	case "HK":
		items := fetchHKStockNews(ctx, ticker, limit)
		lines = append(lines, formatStockItems("港股", items)...)
	case "CN":
		items := fetchCNStockNews(ctx, ticker, code, limit)
		lines = append(lines, formatCNAnnouncements(items)...)
	default:
		items, _ := fetchYahooRSS(ctx, ticker, limit)
		lines = append(lines, formatStockItems("新闻", items)...)
	}
	if len(lines) <= 2 {
		lines = append(lines, "（暂无数据）")
	}
	return strings.Join(lines, "\n"), nil
}

func runWithFallback(ctx context.Context, opts Options, scriptArgs []string, goFn func(context.Context) (string, error)) (string, error) {
	text, err := runScript(ctx, opts, scriptArgs)
	if err == nil {
		return text, nil
	}
	if !isScriptUnavailable(err) {
		return "", err
	}
	if strings.TrimSpace(os.Getenv("GEEGOO_NEWS_DISABLE_GO")) == "1" {
		return "", ErrUnavailable
	}
	return goFn(ctx)
}

func isScriptUnavailable(err error) bool {
	return err == ErrUnavailable || strings.Contains(err.Error(), "news script runner unavailable")
}

func fetchUSIndustryNews(ctx context.Context, limit int) ([]newsItem, error) {
	seen := map[string]bool{}
	var out []newsItem
	for _, feed := range []string{cnbcRSS, djRSS} {
		items, err := fetchRSS(ctx, feed, limit*2)
		if err != nil {
			continue
		}
		for _, it := range items {
			title := strings.TrimSpace(it.Title)
			if title == "" || seen[title] {
				continue
			}
			seen[title] = true
			out = append(out, newsItem{Title: title, Desc: shorten(it.Description, 200), URL: it.Link})
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

type sinaRow struct {
	Title     string `json:"title"`
	Intro     string `json:"intro"`
	Ctime     string `json:"ctime"`
	MediaName string `json:"media_name"`
	URL       string `json:"url"`
}

func fetchSinaRoll(ctx context.Context, keyword string, num int, keywords []string) ([]sinaRow, error) {
	params := url.Values{
		"pageid": {"153"}, "lid": {"2516"}, "k": {keyword},
		"num": {fmt.Sprintf("%d", num)}, "page": {"1"}, "versionNumber": {"1.2.4"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sinaAPI+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var parsed struct {
		Result struct {
			Data []sinaRow `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	rows := parsed.Result.Data
	if len(keywords) == 0 {
		return rows, nil
	}
	var filtered []sinaRow
	for _, row := range rows {
		text := row.Title + row.Intro
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered, nil
}

var hkKeywords = []string{"港股", "恒生", "H股", "港交所", "腾讯", "阿里巴巴", "美团", "小米", "比亚迪"}
var cnKeywords = []string{"A股", "上证", "深证", "沪指", "创业板", "科创板", "沪深", "大盘", "指数"}

func filterHK(rows []sinaRow) []sinaRow {
	var out []sinaRow
	for _, row := range rows {
		text := row.Title + row.Intro
		for _, kw := range hkKeywords {
			if strings.Contains(text, kw) {
				out = append(out, row)
				break
			}
		}
	}
	return out
}

func filterCN(rows []sinaRow) []sinaRow {
	var out []sinaRow
	for _, row := range rows {
		text := row.Title + row.Intro
		for _, kw := range cnKeywords {
			if strings.Contains(text, kw) {
				out = append(out, row)
				break
			}
		}
	}
	return out
}

type newsItem struct {
	Title string
	Desc  string
	URL   string
	Time  string
}

func fetchRSS(ctx context.Context, feedURL string, limit int) ([]rssItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var ch rssChannel
	if err := xml.Unmarshal(raw, &ch); err != nil {
		return nil, err
	}
	if len(ch.Items) > limit {
		ch.Items = ch.Items[:limit]
	}
	return ch.Items, nil
}

func fetchYahooRSS(ctx context.Context, ticker string, limit int) ([]newsItem, error) {
	feed := fmt.Sprintf("https://feeds.finance.yahoo.com/rss/2.0/headline?s=%s&region=US&lang=en-US", url.QueryEscape(ticker))
	items, err := fetchRSS(ctx, feed, limit)
	if err != nil {
		return nil, err
	}
	out := make([]newsItem, 0, len(items))
	for _, it := range items {
		out = append(out, newsItem{Title: it.Title, Desc: shorten(it.Description, 200), URL: it.Link, Time: ""})
	}
	return out, nil
}

func fetchEMAnnouncements(ctx context.Context, stockCode, annType string, limit int) ([]newsItem, error) {
	u := fmt.Sprintf(
		"https://np-anotice-stock.eastmoney.com/api/security/ann?sr=-1&page_size=%d&page_index=1&stock=%s&ann_type=%s",
		limit, stockCode, annType,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Data struct {
			List []struct {
				Title      string `json:"title"`
				NoticeDate string `json:"notice_date"`
				ArtCode    string `json:"art_code"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]newsItem, 0, len(parsed.Data.List))
	for _, row := range parsed.Data.List {
		link := ""
		if row.ArtCode != "" {
			link = "https://data.eastmoney.com/notices/detail/" + row.ArtCode
		}
		out = append(out, newsItem{
			Title: row.Title,
			Time:  strings.TrimSpace(row.NoticeDate),
			URL:   link,
		})
	}
	return out, nil
}

func formatNewsBlock(label string, items []newsItem) string {
	lines := []string{fmt.Sprintf("\n## 【%s】\n", label)}
	for i, it := range items {
		lines = append(lines, fmt.Sprintf("**%d. %s**", i+1, it.Title))
		if it.Desc != "" {
			lines = append(lines, "   "+it.Desc)
		}
		if it.URL != "" {
			lines = append(lines, "   🔗 "+it.URL)
		}
		lines = append(lines, "")
	}
	if len(items) == 0 {
		lines = append(lines, "（暂无数据）")
	}
	return strings.Join(lines, "\n")
}

func formatSinaBlock(label string, rows []sinaRow, limit int) string {
	items := make([]newsItem, 0, limit)
	for _, row := range rows {
		items = append(items, newsItem{Title: row.Title, Desc: shorten(row.Intro, 150), URL: row.URL})
		if len(items) >= limit {
			break
		}
	}
	return formatNewsBlock(label, items)
}

func formatStockItems(section string, items []newsItem) []string {
	if len(items) == 0 {
		return nil
	}
	lines := []string{"\n### " + section + "\n"}
	for i, it := range items {
		lines = append(lines, fmt.Sprintf("**%d. %s**", i+1, it.Title))
		if it.Desc != "" {
			lines = append(lines, "   "+it.Desc)
		}
		if it.URL != "" {
			lines = append(lines, "   🔗 "+it.URL)
		}
		lines = append(lines, "")
	}
	return lines
}

func formatCNAnnouncements(items []newsItem) []string {
	if len(items) == 0 {
		return nil
	}
	lines := []string{"\n### A股公告\n"}
	for i, it := range items {
		lines = append(lines, fmt.Sprintf("**%d. %s**", i+1, it.Title))
		if it.Time != "" {
			lines = append(lines, "   🕐 "+it.Time)
		}
		if it.URL != "" {
			lines = append(lines, "   🔗 "+it.URL)
		}
		lines = append(lines, "")
	}
	return lines
}

func parseStockCode(code string) (market, ticker string) {
	code = strings.TrimSpace(code)
	switch {
	case strings.HasSuffix(code, ".HK"):
		return "HK", strings.TrimLeft(code[:len(code)-3], "0")
	case strings.HasSuffix(code, ".US"), strings.HasSuffix(code, ".O"), strings.HasSuffix(code, ".N"):
		return "US", strings.ToUpper(strings.Split(code, ".")[0])
	case strings.HasSuffix(code, ".SH"):
		return "CN", strings.TrimSuffix(code, ".SH")
	case strings.HasSuffix(code, ".SZ"):
		return "CN", strings.TrimSuffix(code, ".SZ")
	case len(code) == 6 && isDigits(code):
		return "CN", code
	default:
		return "", code
	}
}

func fetchHKStockNews(ctx context.Context, ticker string, limit int) []newsItem {
	seen := map[string]bool{}
	var out []newsItem
	add := func(batch []newsItem) {
		for _, it := range batch {
			title := strings.TrimSpace(it.Title)
			if title == "" || seen[title] {
				continue
			}
			seen[title] = true
			out = append(out, it)
			if len(out) >= limit {
				return
			}
		}
	}
	padded := padHKTicker(ticker)
	if batch, _ := fetchYahooRSS(ctx, padded+".HK", limit); len(batch) > 0 {
		add(batch)
	}
	if len(out) < limit {
		key := strings.TrimLeft(ticker, "0")
		if adr, ok := hkADRMap[key]; ok {
			if batch, _ := fetchYahooRSS(ctx, adr, limit); len(batch) > 0 {
				add(batch)
			}
		}
	}
	if len(out) < limit {
		if rows, _ := fetchSinaRoll(ctx, padded, 40, nil); len(rows) > 0 {
			add(sinaRowsToNews(rows))
		}
	}
	if len(out) < limit {
		if name := hkStockName(ticker); name != "" {
			if rows, _ := fetchSinaRoll(ctx, name, 40, nil); len(rows) > 0 {
				add(sinaRowsToNews(rows))
			}
		}
	}
	return out
}

func fetchCNStockNews(ctx context.Context, ticker, code string, limit int) []newsItem {
	annType := "SZA"
	if strings.HasPrefix(ticker, "6") || strings.HasPrefix(code, "6") {
		annType = "SHA"
	}
	items, _ := fetchEMAnnouncements(ctx, ticker, annType, limit)
	if len(items) >= limit {
		return items
	}
	seen := map[string]bool{}
	for _, it := range items {
		seen[strings.TrimSpace(it.Title)] = true
	}
	if rows, _ := fetchSinaRoll(ctx, ticker, 40, nil); len(rows) > 0 {
		for _, row := range rows {
			title := strings.TrimSpace(row.Title)
			if title == "" || seen[title] {
				continue
			}
			seen[title] = true
			items = append(items, newsItem{
				Title: title,
				Desc:  shorten(row.Intro, 150),
				URL:   row.URL,
			})
			if len(items) >= limit {
				break
			}
		}
	}
	return items
}

func sinaRowsToNews(rows []sinaRow) []newsItem {
	out := make([]newsItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, newsItem{
			Title: row.Title,
			Desc:  shorten(row.Intro, 150),
			URL:   row.URL,
		})
	}
	return out
}

func padHKTicker(ticker string) string {
	ticker = strings.TrimLeft(strings.TrimSpace(ticker), "0")
	if ticker == "" {
		return "00000"
	}
	for len(ticker) < 5 {
		ticker = "0" + ticker
	}
	return ticker
}

var hkADRMap = map[string]string{
	"700": "TCEHY", "9988": "BABA", "3690": "MPNGF", "1810": "XIACF",
	"1211": "BYDDY", "941": "CHL", "388": "HKXCY", "9618": "JD",
}

var hkNameMap = map[string]string{
	"700": "腾讯", "9988": "阿里巴巴", "3690": "美团", "1810": "小米",
	"1211": "比亚迪", "941": "中国移动", "388": "港交所", "9618": "京东",
}

func hkADR(ticker string) string {
	key := strings.TrimLeft(ticker, "0")
	if adr, ok := hkADRMap[key]; ok {
		return adr
	}
	return ticker
}

func hkStockName(ticker string) string {
	key := strings.TrimLeft(ticker, "0")
	return hkNameMap[key]
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func shorten(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}
