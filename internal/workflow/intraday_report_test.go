package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type intradaySynthMock struct {
	res report.IntradaySynthesisResult
	err error
}

func (m *intradaySynthMock) Synthesize(
	ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext,
) (report.SynthesisResult, error) {
	return report.SynthesisResult{}, context.Canceled
}

func (m *intradaySynthMock) SynthesizeMarket(
	ctx context.Context, market, draft string, mc memory.MarketContext, ev []memory.EvidenceRef, template string,
) (report.MarketSynthesisResult, error) {
	return report.MarketSynthesisResult{}, context.Canceled
}

func (m *intradaySynthMock) SynthesizeStockPreMarket(
	ctx context.Context, ws memory.StockWorkspace, draft string, ev []memory.EvidenceRef, mc memory.MarketContext,
	marketReportSummary, template string,
) (report.StockPreMarketSynthesisResult, error) {
	return report.StockPreMarketSynthesisResult{}, context.Canceled
}

func (m *intradaySynthMock) SynthesizeIntraday(
	ctx context.Context, ws memory.StockWorkspace, draft, result, confidence string,
) (report.IntradaySynthesisResult, error) {
	return m.res, m.err
}

func TestBuildIntradayReportContentNoTables(t *testing.T) {
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"601766.SH": {
				Code: "601766.SH", StockName: "中国中车", BotType: "DCA",
				TradeType: "信号买入", PreMarketResult: "long", PreMarketConfidence: "high",
				PreMarketReason: "盘前偏多",
				HourlyPriceAnalysis: `### 一、整体走势概览

| 日期 | 收盘 |
| --- | --- |
| 7/30 | 5.88 |

### 三、综合结论

价格 5.88 元附近震荡偏多。`,
				HourlySignalAnalysis: `### 三、综合多空判定

小时级信号偏多。`,
				CurrentPrice: 5.88, PriceSource: "get_current_price",
				IntradayResult: "buy", IntradayConfidence: "high",
			},
		},
	}
	body := workflow.BuildIntradayReportContent(w, "601766.SH")
	if strings.Contains(body, "| ---") || strings.Contains(body, "| 日期") {
		t.Fatalf("report should not contain tables: %s", body)
	}
	if !strings.Contains(body, "### 小时级价格分析") {
		t.Fatalf("expected hourly section heading: %s", body)
	}
	if strings.Contains(body, "buy") {
		t.Fatalf("expected no English buy in body: %s", body)
	}
}

func TestBuildCreateIntradayReportArgsUsesLLMAndNoTags(t *testing.T) {
	longReason := strings.Repeat("综合盘前偏多、小时级信号与参考价，建议顺势参与并关注量能变化。", 3)
	mock := &intradaySynthMock{
		res: report.IntradaySynthesisResult{
			Summary: "中国中车信号买入，决策买入，置信度高，小时级偏多。",
			Reason:  longReason,
		},
	}
	ctx := workflow.ContextWithSynthesizer(context.Background(), mock)
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"601766.SH": {
				Code: "601766.SH", StockName: "中国中车", BotType: "DCA",
				TradeType: "信号买入", IntradayResult: "buy", IntradayConfidence: "high",
				CurrentPrice: 5.88,
			},
		},
	}
	args := workflow.BuildCreateIntradayReportArgs(ctx, w, "601766.SH")
	if _, ok := args["tags"]; ok {
		t.Fatalf("tags should be omitted: %v", args["tags"])
	}
	if args["summary"] != mock.res.Summary {
		t.Fatalf("summary=%v want LLM", args["summary"])
	}
	reason := args["reason"].(string)
	if strings.Contains(strings.ToLower(reason), "buy") {
		t.Fatalf("reason should not contain buy: %s", reason)
	}
	if len([]rune(reason)) < 80 {
		t.Fatalf("reason too short: %s", reason)
	}
}

func TestBuildIntradayReasonFallbackLocalized(t *testing.T) {
	ctx := context.Background()
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"601766.SH": {
				Code: "601766.SH", StockName: "中国中车", TradeType: "信号买入",
				PreMarketResult: "long", CurrentPrice: 5.88,
				HourlyPriceAnalysis: "偏多",
				IntradayResult: "buy", IntradayConfidence: "high",
			},
		},
	}
	args := workflow.BuildCreateIntradayReportArgs(ctx, w, "601766.SH")
	reason := args["reason"].(string)
	if strings.Contains(strings.ToLower(reason), "buy") {
		t.Fatalf("reason should be localized: %s", reason)
	}
	if !strings.Contains(reason, "买入") {
		t.Fatalf("expected 买入 in reason: %s", reason)
	}
}
