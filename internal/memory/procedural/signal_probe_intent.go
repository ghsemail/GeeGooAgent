package procedural

import "strings"

var signalProbeIntentTokens = []string{
	"信号测试", "策略测试", "测一下有没有", "有没有买卖", "买卖点", "信号密度", "只测信号",
}

// SignalProbeIntent reports whether the user wants signal-only probe rather than PnL backtest.
func SignalProbeIntent(message string) bool {
	if BacktestRunIntent(message) {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	for _, tok := range signalProbeIntentTokens {
		if strings.Contains(msg, strings.ToLower(tok)) {
			return true
		}
	}
	if strings.Contains(msg, "策略") && strings.Contains(msg, "测试") {
		return true
	}
	if strings.Contains(msg, "测试一下") && strings.Contains(msg, "「") {
		return true
	}
	return false
}
