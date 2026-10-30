package chat

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func appendSignalDiagnoseChatBody(b *strings.Builder, flow *Flow) {
	if flow == nil {
		return
	}
	if flow.SignalEval.Method != "" {
		appendSignalDiagnoseWavePnLCards(b, flow.SignalEval)
	}
	if v := strings.TrimSpace(flow.DiagnoseVerdict); v != "" {
		fmt.Fprintf(b, "\n\n**diagnose 结论**：%s", v)
		if s := strings.TrimSpace(flow.DiagnoseSummary); s != "" {
			fmt.Fprintf(b, " — %s", s)
		}
	}
}

// strictWavePnLStats aggregates closed strict episodes (Complete) by win/loss cohort.
// WinPctSum: sum of DirectionReturn on Hit episodes (directional gain).
// LossPctSum: sum of |DirectionReturn| on non-Hit episodes with DirectionReturn < 0 (loss magnitude).
// Flat non-Hit episodes (DirectionReturn == 0) count toward LossCount but add 0 to LossPctSum.
type strictWavePnLStats struct {
	Total      int
	WinCount   int
	WinPctSum  float64
	LossCount  int
	LossPctSum float64
}

func computeStrictWavePnL(details []slots.SignalEpisodeDetail) strictWavePnLStats {
	var s strictWavePnLStats
	for _, ep := range details {
		if !ep.Complete {
			continue
		}
		s.Total++
		if ep.Hit {
			s.WinCount++
			s.WinPctSum += ep.DirectionReturn
			continue
		}
		s.LossCount++
		if ep.DirectionReturn < 0 {
			s.LossPctSum += -ep.DirectionReturn
		}
	}
	return s
}

func appendSignalDiagnoseWavePnLCards(b *strings.Builder, eval slots.SignalEpisodeEval) {
	fmt.Fprintf(b, "\n\n### 波段盈亏（Strict 闭合 · %s）\n", eval.Method)
	fmt.Fprintf(b, "\n涨幅/亏损为各闭合波段 **方向收益**（DirectionReturn）累加：盈利段求和涨幅，亏损段为亏损幅度绝对值之和。\n")

	buy := computeStrictWavePnL(eval.BuyDetails)
	sell := computeStrictWavePnL(eval.SellDetails)
	appendStrictWavePnLCard(b, "买入（做多）", buy)
	appendStrictWavePnLCard(b, "做空（卖出）", sell)

	inc := eval.BuyEpisodes.IncompleteCount + eval.SellEpisodes.IncompleteCount
	if inc > 0 {
		fmt.Fprintf(b, "\n> 另有 Strict **未闭合** %d 段（买 %d / 卖 %d），未计入上表。\n",
			inc, eval.BuyEpisodes.IncompleteCount, eval.SellEpisodes.IncompleteCount)
	}
}

func appendStrictWavePnLCard(b *strings.Builder, sideLabel string, s strictWavePnLStats) {
	fmt.Fprintf(b, "\n#### %s\n\n", sideLabel)
	if s.Total == 0 {
		b.WriteString("- 暂无 Strict 闭合波段。\n")
		return
	}
	fmt.Fprintf(b, "- **总计波段**：%d 段\n", s.Total)
	fmt.Fprintf(b, "- **盈利波段**：%d 段，**累计涨幅** %s\n", s.WinCount, slots.FormatPct(s.WinPctSum))
	fmt.Fprintf(b, "- **亏损波段**：%d 段，**累计亏损** %s\n", s.LossCount, formatLossPctSum(s.LossPctSum))
}

func formatLossPctSum(magnitude float64) string {
	if magnitude <= 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", magnitude*100)
}
