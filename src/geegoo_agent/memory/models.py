"""Pydantic models for working memory."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field


class BotStock(BaseModel):
    code: str
    stock_name: str
    bot_id: str
    bot_name: str
    bot_type: str


class MarketContext(BaseModel):
    indices_done: bool = False
    market_news_done: bool = False
    index_analysis_refs: dict[str, str] = Field(default_factory=dict)
    index_codes_done: list[str] = Field(default_factory=list)
    market_news: dict[str, str] = Field(default_factory=dict)


class StockWorkspace(BaseModel):
    code: str
    stock_name: str = ""
    bot_id: str = ""
    bot_name: str = ""
    bot_type: str = ""
    status: Literal["pending", "collecting", "synthesizing", "reported", "skipped", "failed"] = (
        "pending"
    )
    weekly_analysis_ref: str | None = None
    weekly_parsed: dict[str, Any] | None = None
    synthesis: dict[str, Any] | None = None
    attitude: str | None = None
    capital_flow_summary: str | None = None
    capital_distribution_summary: str | None = None
    report_ref: str | None = None
    report_id: str | None = None
    stock_news_summary: str | None = None


class PreMarketWorking(BaseModel):
    session_id: str
    skill: str = "pre_market"
    phase: Literal["init", "phase_a", "phase_b", "done"] = "init"
    is_trading_day: bool | None = None
    bot_codes: list[BotStock] = Field(default_factory=list)
    market_context: MarketContext = Field(default_factory=MarketContext)
    stocks: dict[str, StockWorkspace] = Field(default_factory=dict)
    artifacts: dict[str, str] = Field(default_factory=dict)
    steps_completed: list[str] = Field(default_factory=list)
    current_stock_code: str | None = None
