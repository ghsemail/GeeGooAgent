package cognition

import "strings"

// PreferAmbiguousClarify reports whether an utterance should route to ambiguous/clarify
// before direct execution: compound intents, bare indicator mentions, or ambiguous quotes.
func PreferAmbiguousClarify(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	return isCompoundAnalyzeBacktest(msg) ||
		isBareIndicatorWithoutAction(msg) ||
		isAmbiguousStockPriceQuery(msg)
}

func isCompoundAnalyzeBacktest(msg string) bool {
	hasAnalysis := hasAny(msg, []string{"分析", "analyze", "评估", "研判"})
	return hasAnalysis && isBacktestRun(msg)
}

var (
	indicatorTokens = []string{
		"macd", "rsi", "sar", "ema", "kdj", "cci", "boll", "bbands",
		"布林带", "均线",
	}
	indicatorActionTokens = []string{
		"回测", "跑回测", "买卖点", "测点", "测信号", "创建", "配置", "优化", "对比", "列出", "监控",
	}
	knowledgeEducationTokens = []string{
		"知识库", "是什么意思", "什么意思", "是什么", "定义", "讲解", "讲讲", "含义",
	}
	catalogListTokens = []string{
		"有哪些", "列出", "列表", "看看我有哪些", "组合信号", "信号策略", "信号组合",
	}
	stockPriceCueTokens = []string{"股价", "价钱", "多少钱", "现价", "价格", "行情", "股票"}
	explicitQuoteTokens = []string{"查现价", "只要当前价", "现价是多少", "查询", "查一下", "查下", "多少港元", "多少美元"}
	explicitAnalysisTokens = []string{"分析", "走势", "技术面", "k线", "趋势", "形态"}
	vagueQualityTokens    = []string{"怎么样", "如何", "好不好", "咋样"}
)

func isBareIndicatorWithoutAction(msg string) bool {
	if !hasAny(msg, indicatorTokens) {
		return false
	}
	if hasAny(msg, knowledgeEducationTokens) || hasAny(msg, catalogListTokens) {
		return false
	}
	if isBacktestRun(msg) || isSignalProbe(msg) {
		return false
	}
	if hasAny(msg, indicatorActionTokens) {
		return false
	}
	// Usage/how-to questions on indicators without a concrete execution verb stay ambiguous.
	if hasAny(msg, []string{"怎么用", "如何用", "用法"}) {
		return true
	}
	return !hasAny(msg, explicitAnalysisTokens)
}

func isAmbiguousStockPriceQuery(msg string) bool {
	if !hasAny(msg, stockPriceCueTokens) {
		return false
	}
	if hasAny(msg, explicitQuoteTokens) {
		return false
	}
	if hasAny(msg, explicitAnalysisTokens) {
		return false
	}
	return hasAny(msg, vagueQualityTokens) || strings.Contains(msg, "股价")
}

func ambiguousClarifyTemplate(msg string) string {
	if isAmbiguousStockPriceQuery(msg) {
		return "stock_quote"
	}
	return ""
}
