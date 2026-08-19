package postmarket

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/decision"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/textutil"
)

// BuildPostMarketSummaryOneLiner mirrors premarket buildStockSummaryOneLiner: a short
// user-facing sentence, not a truncated copy of the report body.
func BuildPostMarketSummaryOneLiner(ws memory.StockWorkspace, sessionBias, vs string) string {
	name := strings.TrimSpace(ws.StockName)
	if name == "" {
		name = strings.TrimSpace(ws.Code)
	}
	bias := decision.LocalizeSessionBias(sessionBias)
	if bias == "" {
		bias = "中性"
	}
	parts := []string{fmt.Sprintf("%s盘后复盘%s", name, bias)}
	if vsLoc := decision.LocalizeVsPreMarket(vs); vsLoc != "" {
		parts = append(parts, fmt.Sprintf("较盘前%s", vsLoc))
	}
	if ws.ChangePct != 0 {
		word := "收涨"
		if ws.ChangePct < 0 {
			word = "收跌"
		}
		parts = append(parts, fmt.Sprintf("%s%.2f%%", word, absChange(ws.ChangePct)))
	}
	return textutil.OneLine(strings.Join(parts, "，")+"。", 200)
}

func absChange(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
