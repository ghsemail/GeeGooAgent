package stockdigest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// Build formats stock pre/post market digests for Feishu IM.
func Build(skill, market string, result workflow.RunResult) string {
	w := result.Working
	date := reportDate(w)
	market = strings.ToUpper(strings.TrimSpace(market))
	if market == "" && w != nil {
		market = strings.ToUpper(strings.TrimSpace(w.Market))
	}

	if w != nil && w.IsTradingDay != nil && !*w.IsTradingDay {
		title := phaseTitle(skill, market)
		return fmt.Sprintf("%s %s 非交易日\n今日无个股报告生成", title, date)
	}

	switch skill {
	case "premarket_stock":
		return buildPremarketStockSummary(market, date, w)
	case "postmarket_stock":
		return buildPostmarketStockSummary(market, date, w)
	default:
		return ""
	}
}

func buildPremarketStockSummary(market, date string, w *memory.PreMarketWorking) string {
	lines := stockLines(w, formatPremarketStock)
	if len(lines) == 0 {
		return fmt.Sprintf("%s\n\n今日无个股盘前报告（无订阅标的或全部跳过）", mdPhaseTitle("premarket_stock", market, date))
	}
	var b strings.Builder
	b.WriteString(mdPhaseTitle("premarket_stock", market, date))
	b.WriteString("\n\n")
	b.WriteString(strings.Join(lines, "\n\n---\n\n"))
	return strings.TrimSpace(b.String())
}

func buildPostmarketStockSummary(market, date string, w *memory.PreMarketWorking) string {
	lines := stockLines(w, formatPostmarketStock)
	if len(lines) == 0 {
		return fmt.Sprintf("%s\n\n今日无个股盘后报告", mdPhaseTitle("postmarket_stock", market, date))
	}
	var b strings.Builder
	b.WriteString(mdPhaseTitle("postmarket_stock", market, date))
	b.WriteString("\n\n")
	b.WriteString(strings.Join(lines, "\n\n---\n\n"))
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

func reportDate(w *memory.PreMarketWorking) string {
	if w != nil && strings.TrimSpace(w.ReportDate) != "" {
		return strings.TrimSpace(w.ReportDate)
	}
	return time.Now().Format("2006-01-02")
}

func phaseTitle(skill, market string) string {
	switch skill {
	case "premarket_stock":
		return fmt.Sprintf("【盘前个股·%s】", market)
	case "postmarket_stock":
		return fmt.Sprintf("【盘后个股·%s】", market)
	default:
		return "【个股报告】"
	}
}
