"""Pydantic schemas for tool payloads and API validation."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, field_validator

ResultType = Literal["long", "short", "neutral"]
SuggestionType = Literal["buy", "sell", "hold"]
ConfidenceType = Literal["high", "medium", "low"]


class PreMarketReportCreate(BaseModel):
    code: str
    stock_name: str
    bot_id: str
    bot_name: str
    bot_type: str
    result: ResultType
    confidence: ConfidenceType
    reason: str
    suggestion: SuggestionType
    report: str
    summary: str = ""
    support: float | None = None
    resistance: float | None = None

    @field_validator("bot_id", "bot_name", "bot_type", "reason", "report", "stock_name", "code")
    @classmethod
    def must_be_non_empty(cls, value: str) -> str:
        if not value or not str(value).strip():
            raise ValueError("must not be empty")
        return str(value).strip()

    def to_api_body(self) -> dict:
        body = self.model_dump(exclude_none=True)
        return body
