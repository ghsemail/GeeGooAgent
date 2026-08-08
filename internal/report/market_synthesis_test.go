package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func TestSynthesizeMarketParsesJSON(t *testing.T) {
	t.Parallel()
	reportMD := "# A股 市场盘前报告\n\n## 一、指数走势\n上证偏强。"
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"report":"` + strings.ReplaceAll(reportMD, "\n", "\\n") + `","result":"long","confidence":"high","summary":"A股盘前偏强，关注量能延续"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	mc := memory.MarketContext{
		IndicesDone:       true,
		MarketNewsDone:    true,
		IndexAnalysisRefs: map[string]string{"000001.SH": "偏强"},
		MarketNews:        map[string]string{"CN": "- 政策偏暖"},
	}
	res, err := synth.SynthesizeMarket(context.Background(), "CN", "# draft", mc, nil, "")
	if err != nil {
		t.Fatalf("synthesize market: %v", err)
	}
	if !strings.Contains(res.Report, "A股") {
		t.Fatalf("report=%q", res.Report)
	}
	if res.Result != "long" || res.Confidence != "high" {
		t.Fatalf("result=%s confidence=%s", res.Result, res.Confidence)
	}
}

func TestSynthesizeMarketRejectsEmptyReport(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"report":"","result":"neutral","confidence":"low","summary":"x"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	_, err := synth.SynthesizeMarket(context.Background(), "CN", "", memory.MarketContext{}, nil, "")
	if err == nil {
		t.Fatal("expected error for empty report")
	}
}
