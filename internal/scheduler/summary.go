package scheduler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/decision"
)

func shouldNotifyJob(job Job) bool {
	if strings.ToLower(strings.TrimSpace(job.Platform)) != "feishu" {
		return false
	}
	switch job.Skill {
	case "premarket_stock", "postmarket_stock":
		return true
	default:
		return false
	}
}

// ShouldNotifyJobForTest exposes notify gating for unit tests.
func ShouldNotifyJobForTest(job Job) bool { return shouldNotifyJob(job) }

// BuildFeishuSummary formats a concise IM digest from workflow working memory.
func BuildFeishuSummary(job Job, result workflow.RunResult) string {
	w := result.Working
	date := reportDate(w)
	market := strings.ToUpper(strings.TrimSpace(job.Market))
	if market == "" && w != nil {
		market = strings.ToUpper(strings.TrimSpace(w.Market))
	}

	verdictLine := verdictLine(result)
	if w != nil && w.IsTradingDay != nil && !*w.IsTradingDay {
		title := phaseTitle(job.Skill, market)
		return fmt.Sprintf("%s %s 非交易日\n- 今日无报告生成\n%s", title, date, verdictLine)
	}

	switch job.Skill {
	case "premarket_stock":
		return buildPremarketStockSummary(market, date, w, verdictLine)
	case "postmarket_stock":
		return buildPostmarketStockSummary(market, date, w, verdictLine)
	default:
		return ""
	}
}

func buildPremarketStockSummary(market, date string, w *memory.PreMarketWorking, verdictLine string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "【盘前准备·%s】%s\n", market, date)
	if w != nil && strings.TrimSpace(w.MarketReportSummary) != "" {
		fmt.Fprintf(&b, "市场：%s\n", memory.OneLine(w.MarketReportSummary, 200))
	} else if w != nil && w.MarketReportResult != "" {
		fmt.Fprintf(&b, "市场：%s/%s\n", w.MarketReportResult, w.MarketReportConfidence)
	}

	lines := stockLines(w, formatPremarketStock)
	if len(lines) == 0 {
		b.WriteString("- 今日无个股盘前报告（无订阅标的或全部跳过）\n")
	} else {
		b.WriteString(strings.Join(lines, "\n---\n"))
		b.WriteByte('\n')
	}
	if verdictLine != "" {
		b.WriteString(verdictLine)
	}
	return strings.TrimSpace(b.String())
}

func buildPostmarketStockSummary(market, date string, w *memory.PreMarketWorking, verdictLine string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "【盘后总结·%s】%s\n", market, date)
	lines := stockLines(w, formatPostmarketStock)
	if len(lines) == 0 {
		b.WriteString("- 今日无个股盘后报告\n")
	} else {
		b.WriteString(strings.Join(lines, "\n---\n"))
		b.WriteByte('\n')
	}
	if verdictLine != "" {
		b.WriteString(verdictLine)
	}
	return strings.TrimSpace(b.String())
}

func stockLines(w *memory.PreMarketWorking, format func(memory.StockWorkspace) string) []string {
	if w == nil || len(w.Stocks) == 0 {
		return nil
	}
	codes := make([]string, 0, len(w.Stocks))
	for code := range w.Stocks {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	var out []string
	for _, code := range codes {
		ws := w.Stocks[code]
		if ws.Status != "reported" {
			continue
		}
		if line := strings.TrimSpace(format(ws)); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func formatPremarketStock(ws memory.StockWorkspace) string {
	name := displayName(ws)
	result := strings.TrimSpace(ws.PreMarketResult)
	if result == "" {
		result = "neutral"
	}
	conf := strings.TrimSpace(ws.PreMarketConfidence)
	suggest := strings.TrimSpace(ws.PreMarketSuggestion)
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", name, ws.Code)
	if result != "" || conf != "" || suggest != "" {
		fmt.Fprintf(&b, " | %s", result)
		if conf != "" {
			b.WriteString("/")
			b.WriteString(conf)
		}
		if suggest != "" {
			fmt.Fprintf(&b, " | %s", suggest)
		}
	}
	if reason := strings.TrimSpace(ws.PreMarketReason); reason != "" {
		fmt.Fprintf(&b, "\n- 判定：%s", memory.OneLine(reason, 120))
	}
	if ws.ReportID != "" {
		fmt.Fprintf(&b, "\n- 报告ID：%s", ws.ReportID)
	}
	return b.String()
}

func formatPostmarketStock(ws memory.StockWorkspace) string {
	name := displayName(ws)
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", name, ws.Code)
	if ws.ChangePct != 0 {
		sign := "+"
		if ws.ChangePct < 0 {
			sign = ""
		}
		fmt.Fprintf(&b, " | %s%.2f%%", sign, ws.ChangePct)
	}
	if bias := decision.LocalizeSessionBias(ws.SessionBias); bias != "" {
		fmt.Fprintf(&b, " | %s", bias)
	}
	if vs := strings.TrimSpace(ws.VsPreMarket); vs != "" && vs != "na" {
		fmt.Fprintf(&b, "\n- 与盘前：%s", vs)
	}
	if pos := strings.TrimSpace(ws.BotLogSummary); pos != "" {
		fmt.Fprintf(&b, "\n- 资金/持仓：%s", memory.OneLine(pos, 100))
	} else if pos := strings.TrimSpace(ws.PositionSummary); pos != "" {
		fmt.Fprintf(&b, "\n- 持仓：%s", memory.OneLine(pos, 100))
	}
	if recap := decision.MarketSummaryFromHourly(ws); recap != "" {
		fmt.Fprintf(&b, "\n- 复盘：%s", memory.OneLine(recap, 120))
	}
	if ws.ReportID != "" {
		fmt.Fprintf(&b, "\n- 报告ID：%s", ws.ReportID)
	}
	return b.String()
}

func displayName(ws memory.StockWorkspace) string {
	if n := strings.TrimSpace(ws.StockName); n != "" {
		return n
	}
	return ws.Code
}

func reportDate(w *memory.PreMarketWorking) string {
	if w != nil && strings.TrimSpace(w.ReportDate) != "" {
		return strings.TrimSpace(w.ReportDate)
	}
	return time.Now().Format("2006-01-02")
}

func verdictLine(result workflow.RunResult) string {
	if result.Supervisor != nil {
		return result.Supervisor.Summary()
	}
	if result.LastError != "" {
		return "verdict=error " + memory.OneLine(result.LastError, 120)
	}
	return ""
}

func phaseTitle(skill, market string) string {
	switch skill {
	case "premarket_stock":
		return fmt.Sprintf("【盘前准备·%s】", market)
	case "postmarket_stock":
		return fmt.Sprintf("【盘后总结·%s】", market)
	default:
		return "【报告】"
	}
}
