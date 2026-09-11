package domaincatalog

// SignalUsageClarifyQuestion is shown when the user asks how to use a vague MACD/signal
// without naming which catalog entry they mean.
const SignalUsageClarifyQuestion = "你想了解哪个 MACD 相关信号？"

// SignalUsageClarifyChoices are catalog-style labels for MACD-related combination signals.
var SignalUsageClarifyChoices = []string{
	"SAR信号搭配MACD直方图趋势",
	"MACD金叉死叉",
}
