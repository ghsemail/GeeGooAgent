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

信号偏多。`,
				CurrentPrice: 5.88, PriceSource: "get_current_price",
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
}

func TestBuildCreateIntradayReportArgsUsesLLMDecision(t *testing.T) {
	longReason := strings.Repeat("综合盘前偏多、小时级信号与参考价，建议顺势参与并关注量能变化。", 3)
	mock := &intradaySynthMock{
		res: report.IntradaySynthesisResult{
			Result:     "buy",
			Confidence: "high",
			Summary:    "中国中车信号买入，决策买入，置信度高，小时级偏多。",
			Reason:     longReason,
		},
	}
	ctx := workflow.ContextWithSynthesizer(context.Background(), mock)
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"601766.SH": {
				Code: "601766.SH", StockName: "中国中车", BotType: "DCA",
				TradeType: "信号买入", CurrentPrice: 5.88,
			},
		},
	}
	args := workflow.BuildCreateIntradayReportArgs(ctx, w, "601766.SH")
	if _, ok := args["tags"]; ok {
		t.Fatalf("tags should be omitted: %v", args["tags"])
	}
	if args["result"] != "buy" {
		t.Fatalf("result=%v want LLM buy", args["result"])
	}
	if args["confidence"] != "high" {
		t.Fatalf("confidence=%v", args["confidence"])
	}
	if args["summary"] != mock.res.Summary {
		t.Fatalf("summary=%v want LLM", args["summary"])
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
			},
		},
	}
	args := workflow.BuildCreateIntradayReportArgs(ctx, w, "601766.SH")
	reason := args["reason"].(string)
	if strings.Contains(strings.ToLower(reason), "buy") {
		t.Fatalf("reason should be localized: %s", reason)
	}
}

func TestIntradayStepsSkipPositionForReminder(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "intraday_stock")
	in := workflow.IntradayInput{
		Code: "00700.HK", BotType: "GRIDReminder", Frequency: "15m", TradeType: "信号买入",
	}
	workflow.SeedIntradayWorking(w, in)
	for _, step := range workflow.IntradayPerStockStepsForWorking(w) {
		if step.Tool == "get_position" {
			t.Fatal("reminder should not run get_position")
		}
		if step.Tool == "get_ticker" {
			t.Fatal("intraday should not run get_ticker")
		}
	}
}

func TestIntradayStepsIncludePositionForTradingBot(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "intraday_stock")
	in := workflow.IntradayInput{
		Code: "00700.HK", BotType: "SmartTrade", Frequency: "15m", TradeType: "信号买入",
	}
	workflow.SeedIntradayWorking(w, in)
	found := false
	for _, step := range workflow.IntradayPerStockStepsForWorking(w) {
		if step.Tool == "get_position" {
			found = true
		}
		if step.Tool == "get_ticker" {
			t.Fatal("intraday should not run get_ticker")
		}
	}
	if !found {
		t.Fatal("trading bot should run get_position")
	}
}

func TestIntradayHardRuleBlocksSellWithoutPosition(t *testing.T) {
	longReason := strings.Repeat("综合盘前偏多、小时级信号与参考价，建议顺势参与并关注量能变化。", 3)
	mock := &intradaySynthMock{
		res: report.IntradaySynthesisResult{
			Result: "sell", Confidence: "high", Summary: "应卖出", Reason: longReason,
		},
	}
	ctx := workflow.ContextWithSynthesizer(context.Background(), mock)
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"601766.SH": {
				Code: "601766.SH", StockName: "中国中车", BotType: "DCA",
				TradeType: "信号卖出", HasPosition: false,
			},
		},
	}
	args := workflow.BuildCreateIntradayReportArgs(ctx, w, "601766.SH")
	if args["result"] != "hold" {
		t.Fatalf("result=%v want hold override", args["result"])
	}
}
