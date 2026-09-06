package slots

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockpick"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var (
	reUSTicker  = regexp.MustCompile(`\b([A-Z]{1,5})(?:\.US)?\b`)
	reHKCode    = regexp.MustCompile(`\b(\d{4,5})(?:\.HK)?\b`)
	reAShare    = regexp.MustCompile(`(?i)\b(\d{6})(?:\.(?:SZ|SH|BJ))?\b`)
	reCJKRun    = regexp.MustCompile(`\p{Han}{2,8}`)
	reIntentPad = regexp.MustCompile(`帮我回测一下|帮我测试一下|帮我回测|回测一下|跑回测|来回测|再回测|测试一下|测一下|看一下|帮我看看|看看|帮我分析一下|分析一下|帮我分析|帮我|就用刚才那套|刚才那套|用现成的来回测|不要新建`)
	reStockAfterIntent = regexp.MustCompile(`(?:分析(?:一下|下)?|回测(?:一下|下)?|查(?:一下|下)?|看(?:一下|下)|测(?:一下|下)?)\s*([\p{Han}]{2,12})`)
)

var tickerStopwords = map[string]struct{}{
	"DAILY": {}, "WEEKLY": {}, "MONTHLY": {}, "YEARLY": {},
	"MACD": {}, "RSI": {}, "SAR": {}, "EMA": {}, "SMA": {}, "KDJ": {}, "BOLL": {}, "ATR": {},
	"US": {}, "HK": {}, "CN": {}, "SZ": {}, "SH": {}, "BJ": {},
	"JSON": {}, "HTTP": {}, "HTML": {}, "SMART": {}, "TRADE": {},
	"AND": {}, "OR": {}, "THE": {}, "FOR": {}, "WITH": {},
}

var cjkStockStops = []string{
	"组合信号", "组合", "信号", "策略", "回测", "收益", "收益率", "回撤", "成交",
	"趋势", "直方图", "金叉", "死叉", "抛物线", "共振", "哪些", "现在", "标的",
	"股票", "一下", "请问", "帮我", "测试", "频率", "支持", "配套", "规则",
	"买入", "卖出", "简介", "当前", "全部", "三种", "共有", "怎么样", "分析",
	"有没有", "买卖点", "买卖", "买点", "卖点", "技术面", "价格", "K线", "线图", "蜡烛",
}

// Longest-first suffixes glued to a company name in probe/backtest utterances.
var stockIntentSuffixes = []string{
	"有没有买卖点", "有没有买卖", "有没有买点", "有没有卖点",
	"有没有买", "有没有卖", "有没有", "买卖点", "买点", "卖点", "买卖",
}

var knownStockAliases = []string{
	"小米", "腾讯", "茅台", "苹果", "特斯拉", "英伟达", "微软", "谷歌", "阿里", "拼多多",
}

// ExtractExplicitStockReference returns a stock only when the user clearly names one
// (alias, code, or a multi-character Chinese name). Follow-up phrases like「分析 K 线」return empty.
func ExtractExplicitStockReference(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	for _, key := range knownStockAliases {
		if strings.Contains(msg, key) {
			return key
		}
	}
	if m := reAShare.FindStringSubmatch(msg); len(m) > 1 {
		if !(strings.Contains(msg, "资金") || strings.Contains(msg, "本金") || strings.Contains(strings.ToLower(msg), "fund")) {
			return strings.ToUpper(m[1])
		}
	}
	if m := reHKCode.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	upper := strings.ToUpper(msg)
	if m := reUSTicker.FindStringSubmatch(upper); len(m) > 1 {
		tok := m[1]
		if _, stop := tickerStopwords[tok]; !stop {
			if len(tok) == 1 && containsHanRun(msg) {
				// skip
			} else {
				return tok
			}
		}
	}
	if q := extractStockAfterIntentVerb(msg); q != "" {
		return q
	}
	if q := extractTrailingCJKStock(msg); q != "" {
		return q
	}
	stripped := reIntentPad.ReplaceAllString(msg, " ")
	for _, run := range reCJKRun.FindAllString(stripped, -1) {
		if q, ok := acceptStockCandidate(run); ok {
			return q
		}
	}
	return ""
}

// ExtractStockQuery pulls a stock name or code fragment from user text.
func ExtractStockQuery(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	for _, key := range knownStockAliases {
		if strings.Contains(msg, key) {
			return key
		}
	}
	if m := reAShare.FindStringSubmatch(msg); len(m) > 1 {
		if !(strings.Contains(msg, "资金") || strings.Contains(msg, "本金") || strings.Contains(strings.ToLower(msg), "fund")) {
			return strings.ToUpper(m[1])
		}
	}
	if m := reHKCode.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	upper := strings.ToUpper(msg)
	if m := reUSTicker.FindStringSubmatch(upper); len(m) > 1 {
		tok := m[1]
		if _, stop := tickerStopwords[tok]; !stop {
			// 「K线图」「KDJ」等中文句里的单字母 Latin 不是美股代码。
			if len(tok) == 1 && containsHanRun(msg) {
				// fall through to CJK extraction
			} else {
				return tok
			}
		}
	}
	if q := extractStockAfterIntentVerb(msg); q != "" {
		return q
	}
	if q := extractTrailingCJKStock(msg); q != "" {
		return q
	}
	stripped := reIntentPad.ReplaceAllString(msg, " ")
	cjk := reCJKRun.FindAllString(stripped, -1)
	for i := len(cjk) - 1; i >= 0; i-- {
		if q, ok := acceptStockCandidate(cjk[i]); ok {
			return q
		}
	}
	return ""
}

func extractStockAfterIntentVerb(msg string) string {
	matches := reStockAfterIntent.FindAllStringSubmatch(msg, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if len(matches[i]) < 2 {
			continue
		}
		if q, ok := acceptStockCandidate(matches[i][1]); ok {
			return q
		}
	}
	return ""
}

// extractTrailingCJKStock reads the trailing Han company-name fragment at the end of the utterance.
func extractTrailingCJKStock(msg string) string {
	runes := []rune(strings.TrimSpace(msg))
	var buf []rune
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		if r >= '\u4e00' && r <= '\u9fff' {
			buf = append([]rune{r}, buf...)
			continue
		}
		if len(buf) > 0 {
			break
		}
	}
	if q, ok := acceptStockCandidate(string(buf)); ok {
		return q
	}
	return ""
}

// normalizeStockCandidate strips intent glue words accidentally merged with a company name.
func normalizeStockCandidate(raw string) string {
	s := trimTrailingStockParticles(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "有没有"); idx > 0 {
		prefix := strings.TrimSpace(s[:idx])
		if len([]rune(prefix)) >= 2 {
			s = prefix
		}
	}
	for {
		changed := false
		for _, suf := range stockIntentSuffixes {
			if strings.HasSuffix(s, suf) && len([]rune(s)) > len([]rune(suf)) {
				s = strings.TrimSpace(strings.TrimSuffix(s, suf))
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	for _, prefix := range []string{"看看", "看下", "看一下", "分析", "查", "看", "测"} {
		if strings.HasPrefix(s, prefix) && len([]rune(s)) > len([]rune(prefix)) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return strings.TrimSpace(s)
}

func acceptStockCandidate(raw string) (string, bool) {
	q := normalizeStockCandidate(raw)
	if q == "" {
		return "", false
	}
	n := len([]rune(q))
	if n < 2 {
		return "", false
	}
	if isCJKStockStop(q) {
		return "", false
	}
	if _, stop := tickerStopwords[strings.ToUpper(q)]; stop {
		return "", false
	}
	if n == 2 {
		for _, alias := range knownStockAliases {
			if q == alias {
				return q, true
			}
		}
		return "", false
	}
	return q, true
}

func trimTrailingStockParticles(s string) string {
	particles := []string{"吧", "呢", "啊", "吗", "的", "了", "呀"}
	for {
		changed := false
		for _, p := range particles {
			if strings.HasSuffix(s, p) && len([]rune(s)) > len([]rune(p)) {
				s = strings.TrimSuffix(s, p)
				changed = true
			}
		}
		if !changed {
			return strings.TrimSpace(s)
		}
	}
}

// IsLikelyStockUtterance reports bare stock names like「中际旭创呢」.
func IsLikelyStockUtterance(msg string) bool {
	q := ExtractStockQuery(msg)
	if q == "" {
		return false
	}
	return LooksLikeStockQuery(q)
}

func containsHanRun(s string) bool {
	for _, r := range s {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

func isCJKStockStop(s string) bool {
	for _, tok := range cjkStockStops {
		if s == tok || strings.Contains(s, tok) {
			return true
		}
	}
	return false
}

// SanitizeStockQuery keeps ExtractStockQuery output only when it plausibly names a ticker.
func SanitizeStockQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" || !LooksLikeStockQuery(q) {
		return ""
	}
	return q
}

// LooksLikeStockQuery rejects frequency words and indicator tokens mistaken as tickers.
func LooksLikeStockQuery(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	if _, stop := tickerStopwords[strings.ToUpper(q)]; stop {
		return false
	}
	if isCJKStockStop(q) {
		return false
	}
	return true
}

// AutoPickStockItem picks a single catalog row without blocking clarify when confident.
func AutoPickStockItem(query string, items []map[string]any) (map[string]any, bool) {
	return stockpick.AutoPick(query, items)
}

// ResolveStock runs search_code and clarifies only when auto-pick cannot disambiguate.
func ResolveStock(
	ctx context.Context,
	toolCtx tools.Context,
	runTool func(context.Context, tools.CallRequest, tools.Context) tools.Result,
	query string,
) (code, name, market string, err error) {
	res := runTool(ctx, tools.CallRequest{Name: "search_code", Arguments: map[string]any{"regex": query}}, toolCtx)
	if res.Status != tools.StatusOK {
		return "", "", "", fmt.Errorf("search_code 失败：%s", res.Summary)
	}
	items := CatalogItems(res.Data)
	if len(items) == 0 {
		return "", "", "", fmt.Errorf("未找到标的「%s」，请换名称或代码", query)
	}
	row, err := pickStockRow(ctx, toolCtx, query, items)
	if err != nil {
		return "", "", "", err
	}
	code = strings.TrimSpace(fmt.Sprint(row["code"]))
	name = strings.TrimSpace(fmt.Sprint(row["name"]))
	market = strings.TrimSpace(fmt.Sprint(row["market"]))
	if code == "" {
		return "", "", "", fmt.Errorf("search_code 未返回有效 code")
	}
	return code, name, market, nil
}

func pickStockRow(ctx context.Context, toolCtx tools.Context, query string, items []map[string]any) (map[string]any, error) {
	if picked, ok := stockpick.AutoPick(query, items); ok {
		return picked, nil
	}
	limit := minInt(len(items), 4)
	choices := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		it := items[i]
		choices = append(choices, fmt.Sprintf("%v %v", it["code"], it["name"]))
	}
	question := "找到多个标的，请选择："
	if q := strings.TrimSpace(query); q != "" {
		question = fmt.Sprintf("搜索「%s」找到多个标的，请选择：", q)
	}
	if toolCtx.ClarifyFn == nil {
		return nil, fmt.Errorf("%s %s", question, strings.Join(choices, " / "))
	}
	answer, ok := toolCtx.ClarifyFn(ctx, question, choices)
	if !ok {
		return nil, fmt.Errorf("请选择要操作的标的")
	}
	for _, it := range items {
		label := fmt.Sprintf("%v %v", it["code"], it["name"])
		if label == answer || strings.Contains(answer, fmt.Sprint(it["code"])) {
			return it, nil
		}
	}
	return nil, fmt.Errorf("未匹配所选标的「%s」", answer)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
