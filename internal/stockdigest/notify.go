package stockdigest

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// ShouldNotify reports whether a stock pre/post market run should push Feishu digest.
func ShouldNotify(skill, market string, result workflow.RunResult) bool {
	return NotifySkipReason(skill, market, result) == ""
}

// NotifySkipReason returns empty string when notify should proceed; otherwise a short reason.
func NotifySkipReason(skill, market string, result workflow.RunResult) string {
	if !result.OK() {
		return "workflow_not_completed"
	}
	if result.Supervisor != nil && result.Supervisor.Verdict != workflow.VerdictPass {
		return "supervisor_" + string(result.Supervisor.Verdict)
	}
	w := result.Working
	if w == nil {
		return "missing_working"
	}
	if w.IsTradingDay != nil && !*w.IsTradingDay {
		return "non_trading_day"
	}
	for code, ws := range w.Stocks {
		if ws.Status == "failed" {
			return "stock_failed:" + code
		}
	}
	if !hasNewlyReportedStock(w) {
		return "no_new_reports"
	}
	format := formatPremarketStock
	if skill == "postmarket_stock" {
		format = formatPostmarketStock
	}
	if len(stockLines(w, format)) == 0 {
		return "no_deliverable_reports"
	}
	return ""
}

// HasDeliverableContent is true when the digest would contain at least one stock section.
func HasDeliverableContent(skill string, result workflow.RunResult) bool {
	w := result.Working
	if w == nil {
		return false
	}
	format := formatPremarketStock
	if skill == "postmarket_stock" {
		format = formatPostmarketStock
	}
	return len(stockLines(w, format)) > 0
}

func hasNewlyReportedStock(w *memory.PreMarketWorking) bool {
	if w == nil {
		return false
	}
	for _, ws := range w.Stocks {
		if ws.Status == "reported" {
			return true
		}
	}
	return false
}

// IsEmptyDigestMessage detects placeholder texts that must not be pushed.
func IsEmptyDigestMessage(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	for _, marker := range []string{
		"今日无个股盘后报告",
		"今日无个股盘前报告",
		"今日无个股报告生成",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
