package strategydev

import (
	"fmt"
	"strings"
)

// SignalDiagnoseProbeHitLine formats probe buy/sell hit counts for chat and reports.
func SignalDiagnoseProbeHitLine(longOnly bool, buyHits, sellHits int) string {
	if !longOnly {
		return fmt.Sprintf("买入 %d 次 / 卖出 %d 次", buyHits, sellHits)
	}
	if sellHits > 0 {
		return fmt.Sprintf("只做多 · 买入 %d 次 · 平多卖出信号 %d 次（非做空 Episode）", buyHits, sellHits)
	}
	return fmt.Sprintf("只做多 · 买入 %d 次", buyHits)
}

// LabSignalSideTriggerHeading labels the signal leg in lab JEV state (long_only: 平多卖出 vs 做空).
func LabSignalSideTriggerHeading(signalSide string, longOnly bool) string {
	side := strings.ToLower(strings.TrimSpace(signalSide))
	switch side {
	case "buy":
		return "信号侧：买入触发"
	case "sell":
		if longOnly {
			return "信号侧：平多卖出信号"
		}
		return "信号侧：卖出触发"
	default:
		return ""
	}
}
