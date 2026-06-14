"""Backward-compatible alias — all GeeGoo MCP routes use port 5700."""

from geegoo_agent.clients.geegoo_bot import (
    BotYesterdayAttitude,
    CapitalDistributionData,
    CapitalFlowItem,
    CapitalFlowPeriod,
    GeeGooBotClient,
    McpAnalysisData,
    McpAnalysisPeriod,
    PreMarketReportResult,
    TradingDayData,
    TradingDayResponse,
    UserBotCode,
)

MarketClient = GeeGooBotClient
McpAnalysisResult = McpAnalysisData
