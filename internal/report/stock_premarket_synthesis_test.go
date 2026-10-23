package report_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func validStockPremarketJSON(report string) string {
	reason := strings.Repeat("综合证据与Bot态度，盘前方向需结合量能与大盘验证。", 4)
	return `{"report":"` + strings.ReplaceAll(report, "\n", "\\n") + `","result":"long","confidence":"high","reason":"` + reason + `","suggestion":"buy","summary":"盘前偏多，关注开盘量价"}`
}

func TestSynthesizeStockPreMarketRetriesOnEmptyContent(t *testing.T) {
	t.Parallel()
	reportMD := "## 市场背景\n\n腾讯控股盘前观察。"
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: ""},
		{Content: validStockPremarketJSON(reportMD)},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 3, RetryWait: 0})
	gateway.SetSleep(func(time.Duration) {})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"}
	res, err := synth.SynthesizeStockPreMarket(context.Background(), ws, "draft", nil, memory.MarketContext{}, "市场摘要", "")
	if err != nil {
		t.Fatalf("synthesize stock premarket: %v", err)
	}
	if !strings.Contains(res.Report, "腾讯") {
		t.Fatalf("report=%q", res.Report)
	}
}

func TestSynthesizeStockPreMarketRetriesOnBadJSON(t *testing.T) {
	t.Parallel()
	reportMD := "## 市场背景\n\n正文。"
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"report":"x","result":"long"`},
		{Content: validStockPremarketJSON(reportMD)},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 3, RetryWait: 0})
	gateway.SetSleep(func(time.Duration) {})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"}
	_, err := synth.SynthesizeStockPreMarket(context.Background(), ws, "draft", nil, memory.MarketContext{}, "", "")
	if err != nil {
		t.Fatalf("synthesize stock premarket: %v", err)
	}
}
