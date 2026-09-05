package slots

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var (
	reUSTicker  = regexp.MustCompile(`\b([A-Z]{1,5})(?:\.US)?\b`)
	reHKCode    = regexp.MustCompile(`\b(\d{4,5})(?:\.HK)?\b`)
	reAShare    = regexp.MustCompile(`(?i)\b(\d{6})(?:\.(?:SZ|SH|BJ))?\b`)
	reCJKRun    = regexp.MustCompile(`\p{Han}{2,8}`)
	reIntentPad = regexp.MustCompile(`帮我回测一下|帮我测试一下|帮我回测|回测一下|跑回测|来回测|再回测|测试一下|测一下|看一下|分析一下|帮我分析|分析一下|就用刚才那套|刚才那套|用现成的来回测|不要新建`)
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
	"有没有", "买卖点", "买卖",
}

var knownStockAliases = []string{
	"小米", "腾讯", "茅台", "苹果", "特斯拉", "英伟达", "微软", "谷歌", "阿里", "拼多多",
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
			return tok
		}
	}
	stripped := reIntentPad.ReplaceAllString(msg, " ")
	cjk := reCJKRun.FindAllString(stripped, -1)
	for i := len(cjk) - 1; i >= 0; i-- {
		if !isCJKStockStop(cjk[i]) {
			return cjk[i]
		}
	}
	return ""
}

// IsLikelyStockUtterance reports bare stock names like「中际旭创呢」.
func IsLikelyStockUtterance(msg string) bool {
	q := ExtractStockQuery(msg)
	if q == "" {
		return false
	}
	return LooksLikeStockQuery(q)
}

func isCJKStockStop(s string) bool {
	for _, tok := range cjkStockStops {
		if s == tok || strings.Contains(s, tok) {
			return true
		}
	}
	return false
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

// ResolveStock runs search_code and clarifies when multiple catalog rows match.
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
	if len(items) == 1 {
		return items[0], nil
	}
	if picked, ok := pickStockRowByCode(items, query); ok {
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

func pickStockRowByCode(items []map[string]any, query string) (map[string]any, bool) {
	q := strings.ToUpper(strings.TrimSpace(query))
	if q == "" || len(items) == 0 {
		return nil, false
	}
	qDigits := strings.TrimLeft(q, "0")
	var containsMatches []map[string]any
	for _, row := range items {
		code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["code"])))
		if code == "" {
			continue
		}
		if code == q {
			return row, true
		}
		codeBase := strings.TrimSuffix(code, ".HK")
		codeBase = strings.TrimSuffix(codeBase, ".SH")
		codeBase = strings.TrimSuffix(codeBase, ".SZ")
		codeBase = strings.TrimSuffix(codeBase, ".US")
		if strings.HasSuffix(q, ".HK") || strings.HasSuffix(q, ".SH") || strings.HasSuffix(q, ".SZ") || strings.HasSuffix(q, ".US") {
			if code == q || strings.HasPrefix(code, q) {
				return row, true
			}
		}
		if qDigits != "" && (codeBase == qDigits || strings.HasSuffix(codeBase, qDigits) || strings.HasSuffix(qDigits, strings.TrimLeft(codeBase, "0"))) {
			return row, true
		}
		if strings.Contains(code, q) || strings.Contains(q, codeBase) {
			containsMatches = append(containsMatches, row)
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches[0], true
	}
	return nil, false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
