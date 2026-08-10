package workflow

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/decision"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/intraday"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/postmarket"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/premarket"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/step"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/synthctx"
)

// Step is one workflow step (see internal/workflow/step).
type Step = step.Step

// SynthesizerProvider abstracts report synthesis (see internal/workflow/synthctx).
type SynthesizerProvider = synthctx.SynthesizerProvider

// ContextWithSynthesizer attaches an optional report synthesizer to ctx.
func ContextWithSynthesizer(ctx context.Context, synth SynthesizerProvider) context.Context {
	return synthctx.ContextWithSynthesizer(ctx, synth)
}

// SynthesizerFrom returns the synthesizer attached to ctx, if any.
func SynthesizerFrom(ctx context.Context) SynthesizerProvider {
	return synthctx.From(ctx)
}

// Market codes for pre-market workflows.
const (
	MarketCN = premarket.MarketCN
	MarketHK = premarket.MarketHK
	MarketUS = premarket.MarketUS
)

// Premarket market workflow API.
var (
	MarketFromCode                     = premarket.MarketFromCode
	NormalizeMarket                    = premarket.NormalizeMarket
	MarketPhaseSteps                   = premarket.MarketPhaseSteps
	StockPhaseASteps                   = premarket.StockPhaseASteps
	BuildMarketReportContent           = premarket.BuildMarketReportContent
	BuildCreateMarketReportArgs        = premarket.BuildCreateMarketReportArgs
	BuildCreateMarketReportArgsContext = premarket.BuildCreateMarketReportArgsContext
	ExpectedIndexCount                 = premarket.ExpectedIndexCount
	ReportDateFor                      = premarket.ReportDateFor
	IsBackfillRun                      = premarket.IsBackfillRun
	SeedReportDate                     = premarket.SeedReportDate
	SeedMarketWorking                  = premarket.SeedMarketWorking
	MarketSynthesizerFrom              = premarket.MarketSynthesizerFrom
)

// Premarket stock workflow API.
var (
	PerStockSteps                 = premarket.PerStockSteps
	BuildReportContent            = premarket.BuildReportContent
	BuildCreateReportArgs         = premarket.BuildCreateReportArgs
	BuildCreateReportArgsContext  = premarket.BuildCreateReportArgsContext
	StockPreMarketSynthesizerFrom = premarket.StockPreMarketSynthesizerFrom
)

// Intraday workflow API.
var (
	DefaultIntradayInput            = intraday.DefaultIntradayInput
	IntradayInputFromEnv            = intraday.IntradayInputFromEnv
	SeedIntradayWorking             = intraday.SeedIntradayWorking
	IntradayPhaseASteps             = intraday.IntradayPhaseASteps
	IntradayPerStockSteps           = intraday.IntradayPerStockSteps
	IntradayPerStockStepsForWorking = intraday.IntradayPerStockStepsForWorking
	BuildIntradayReportContent      = intraday.BuildIntradayReportContent
	BuildCreateIntradayReportArgs   = intraday.BuildCreateIntradayReportArgs
	IntradaySynthesizerFrom         = intraday.IntradaySynthesizerFrom
	IntradayHourlySummarizerFrom    = intraday.IntradayHourlySummarizerFrom
)

// Post-market workflow API.
var (
	PostMarketPhaseASteps                = postmarket.PostMarketPhaseASteps
	PostMarketPerStockSteps              = postmarket.PostMarketPerStockSteps
	BuildPostMarketReportContent         = postmarket.BuildPostMarketReportContent
	BuildCreateStockPostmarketReportArgs = postmarket.BuildCreateStockPostmarketReportArgs
	PostMarketSynthesizerFrom            = postmarket.PostMarketSynthesizerFrom
)

// Decision helpers shared across phases.
var (
	DecideIntraday           = decision.DecideIntraday
	SessionBiasFromChangePct = decision.SessionBiasFromChangePct
	VsPreMarket              = decision.VsPreMarket
	BotLogType               = decision.BotLogType
	MarketSummaryFromHourly  = decision.MarketSummaryFromHourly
	TradeSummaryFromBotLog   = decision.TradeSummaryFromBotLog
	ExperienceSummaryDefault = decision.ExperienceSummaryDefault
)

// Exported bundle / input types for callers.
type (
	MarketReportBundle                = premarket.MarketReportBundle
	StockReportBundle                 = premarket.StockReportBundle
	IntradayInput                     = intraday.IntradayInput
	IntradayBundle                    = intraday.IntradayBundle
	MarketSynthesizerProvider         = premarket.MarketSynthesizerProvider
	StockPreMarketSynthesizerProvider = premarket.StockPreMarketSynthesizerProvider
	IntradaySynthesizerProvider       = intraday.IntradaySynthesizerProvider
	IntradayHourlySummarizerProvider  = intraday.IntradayHourlySummarizerProvider
	PostMarketSynthesizerProvider     = postmarket.PostMarketSynthesizerProvider
)
