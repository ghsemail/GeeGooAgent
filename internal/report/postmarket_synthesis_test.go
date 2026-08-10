package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func TestSynthesizePostMarketSummariesParsesJSON(t *testing.T) {
	t.Parallel()
	market := strings.Repeat("腾讯今日收涨约百分之零点五，盘面中性震荡，小时级量价显示空方占优但跌幅有限。", 2)
	trade := "当日网格机器人无成交，持仓与策略状态维持不变，关注次日信号触发。"
	exp := strings.Repeat("盘前中性与收盘中性一致，复盘应继续核对网格挂单密度与波动是否匹配，避免在震荡市过度加仓。", 2)
	sum := "腾讯收涨中性，与盘前一致，网格无成交。"
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"market_summary":"` + market + `","trade_summary":"` + trade + `","experience_summary":"` + exp + `","summary":"` + sum + `"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股", BotType: "GRID", ChangePct: 0.54}
	res, err := synth.SynthesizePostMarketSummaries(context.Background(), ws, "## draft", "neutral", "aligned", "rule-m", "rule-t", "rule-e")
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if !strings.Contains(res.MarketSummary, "腾讯") {
		t.Fatalf("market=%q", res.MarketSummary)
	}
	if !strings.Contains(res.TradeSummary, "网格") {
		t.Fatalf("trade=%q", res.TradeSummary)
	}
	if res.Summary != sum {
		t.Fatalf("summary=%q", res.Summary)
	}
}

func TestSynthesizePostMarketSummariesRejectsEmpty(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"market_summary":"x","trade_summary":"","experience_summary":"y","summary":"z"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	_, err := synth.SynthesizePostMarketSummaries(context.Background(), memory.StockWorkspace{}, "", "neutral", "na", "", "", "")
	if err == nil {
		t.Fatal("expected empty-field error")
	}
}
