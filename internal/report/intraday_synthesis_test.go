package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func TestSynthesizeIntradayParsesJSON(t *testing.T) {
	t.Parallel()
	longReason := strings.Repeat("综合盘前观点与小时级技术面，当前信号偏多，建议顺势参与并关注量价配合与回撤风险。", 2)
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"summary":"中国中车信号买入，决策买入，置信度高。","reason":"` + longReason + `"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{
		Code: "601766.SH", StockName: "中国中车", TradeType: "信号买入",
	}
	res, err := synth.SynthesizeIntraday(context.Background(), ws, "## draft", "buy", "high")
	if err != nil {
		t.Fatalf("synthesize intraday: %v", err)
	}
	if !strings.Contains(res.Summary, "中国中车") {
		t.Fatalf("summary=%q", res.Summary)
	}
	if strings.Contains(strings.ToLower(res.Reason), "buy") {
		t.Fatalf("reason should be localized: %q", res.Reason)
	}
}

func TestSynthesizeIntradayRejectsShortReason(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"summary":"ok","reason":"太短"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	_, err := synth.SynthesizeIntraday(context.Background(), memory.StockWorkspace{}, "", "buy", "high")
	if err == nil {
		t.Fatal("expected error for short reason")
	}
}
