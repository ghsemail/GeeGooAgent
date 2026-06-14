"""Unit tests for PreMarketReportCreate validation."""

from __future__ import annotations

import pytest
from pydantic import ValidationError

from geegoo_agent.tools.schemas import PreMarketReportCreate


def _valid_payload() -> dict:
    return {
        "code": "00700.HK",
        "stock_name": "腾讯控股",
        "bot_id": "bot-1",
        "bot_name": "DCA",
        "bot_type": "DCA",
        "result": "long",
        "confidence": "high",
        "reason": "周线支撑有效",
        "suggestion": "buy",
        "report": "完整报告内容",
    }


@pytest.mark.unit
def test_valid_report_passes() -> None:
    report = PreMarketReportCreate.model_validate(_valid_payload())
    assert report.bot_id == "bot-1"
    assert report.to_api_body()["result"] == "long"


@pytest.mark.unit
def test_empty_bot_id_rejected() -> None:
    payload = _valid_payload()
    payload["bot_id"] = ""
    with pytest.raises(ValidationError):
        PreMarketReportCreate.model_validate(payload)


@pytest.mark.unit
def test_invalid_result_enum_rejected() -> None:
    payload = _valid_payload()
    payload["result"] = "up"
    with pytest.raises(ValidationError):
        PreMarketReportCreate.model_validate(payload)


@pytest.mark.unit
def test_empty_report_rejected() -> None:
    payload = _valid_payload()
    payload["report"] = "   "
    with pytest.raises(ValidationError):
        PreMarketReportCreate.model_validate(payload)
