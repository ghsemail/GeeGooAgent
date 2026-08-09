package workflow

import (
	"context"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// IntradayHourlySummarizerProvider condenses raw MCP hourly text for report bodies.
type IntradayHourlySummarizerProvider interface {
	SummarizeIntradayHourly(
		ctx context.Context,
		ws memory.StockWorkspace,
		priceRaw, signalRaw, klineRaw string,
	) (report.IntradayHourlySummary, error)
}

// IntradayHourlySummarizerFrom returns an hourly summarizer from context when wired.
func IntradayHourlySummarizerFrom(ctx context.Context) IntradayHourlySummarizerProvider {
	synth := SynthesizerFrom(ctx)
	if synth == nil {
		return nil
	}
	p, ok := synth.(IntradayHourlySummarizerProvider)
	if !ok {
		return nil
	}
	return p
}

func resolveIntradayHourlySections(ctx context.Context, ws memory.StockWorkspace) report.IntradayHourlySummary {
	fallback := report.IntradayHourlySummary{
		Price:  stockfmt.FormatIntradayHourlySection(ws.HourlyPriceAnalysis, "暂无小时级价格分析。"),
		Signal: stockfmt.FormatIntradaySignalSection(ws.HourlySignalAnalysis),
		Kline:  stockfmt.FormatIntradayHourlySection(ws.HourlyKlineAnalysis, "暂无小时级 K 线分析。"),
	}
	if strings.TrimSpace(ws.HourlyPriceAnalysis+ws.HourlySignalAnalysis+ws.HourlyKlineAnalysis) == "" {
		return fallback
	}
	if synth := IntradayHourlySummarizerFrom(ctx); synth != nil {
		if res, err := synth.SummarizeIntradayHourly(ctx, ws, ws.HourlyPriceAnalysis, ws.HourlySignalAnalysis, ws.HourlyKlineAnalysis); err == nil {
			if v := strings.TrimSpace(res.Price); v != "" {
				fallback.Price = v
			}
			if v := strings.TrimSpace(res.Signal); v != "" {
				fallback.Signal = v
			}
			if v := strings.TrimSpace(res.Kline); v != "" {
				fallback.Kline = v
			}
		}
	}
	return fallback
}
