package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func TestSummarizeIntradayHourlyParsesJSON(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"price_analysis":"**整体走势概述**\n\n近5日震荡偏弱。","signal_analysis":"**综合判定**\n\n短线偏空。","kline_analysis":""}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股"}
	res, err := synth.SummarizeIntradayHourly(context.Background(), ws, "raw price", "raw signal", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Price, "整体走势概述") || strings.Contains(res.Price, "8/3：") {
		t.Fatalf("unexpected price summary: %s", res.Price)
	}
	if !strings.Contains(res.Signal, "综合判定") {
		t.Fatalf("unexpected signal summary: %s", res.Signal)
	}
}
