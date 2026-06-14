"""Unit tests for LLM structured tasks."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from geegoo_agent.exceptions import LLMTaskError
from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.types import LLMResponse, TokenUsage
from geegoo_agent.memory.models import BotStock, PreMarketWorking, StockWorkspace
from geegoo_agent.runtime.llm_tasks import (
    PreMarketReportContext,
    PreMarketSynthesis,
    WeeklyAnalysisParsed,
    enrich_stock_with_llm,
    parse_weekly_analysis,
    synthesize_pre_market_report,
)
from geegoo_agent.runtime.pre_market_report import build_create_report_args, build_report_content

FIXTURES = Path(__file__).resolve().parents[1] / "fixtures" / "llm"


class FakeProvider:
    def __init__(self, model: str, responses: list[str]) -> None:
        self.model = model
        self._responses = list(responses)
        self.calls = 0

    def chat(self, messages, tools, *, temperature, max_tokens) -> LLMResponse:
        self.calls += 1
        content = self._responses.pop(0)
        return LLMResponse(
            content=content,
            tool_calls=[],
            usage=TokenUsage(prompt_tokens=10, completion_tokens=20, model=self.model),
        )


def _load_fixture(name: str) -> str:
    return json.dumps(json.loads((FIXTURES / name).read_text(encoding="utf-8")))


def _mock_gateway(responses: list[str]) -> ModelGateway:
    provider = FakeProvider("gpt-4o", responses)
    return ModelGateway(provider, CostManager(), GatewayConfig(max_retries=1))


def _sample_working() -> PreMarketWorking:
    return PreMarketWorking(
        session_id="sess-llm",
        bot_codes=[
            BotStock(
                code="00700.HK",
                stock_name="腾讯控股",
                bot_id="bot-1",
                bot_name="test-bot",
                bot_type="DCA",
            )
        ],
        stocks={
            "00700.HK": StockWorkspace(
                code="00700.HK",
                stock_name="腾讯控股",
                bot_id="bot-1",
                bot_name="test-bot",
                bot_type="DCA",
                weekly_analysis_ref="周线支撑380阻力420",
                attitude="bullish",
                capital_flow_summary="main_in_flow=100",
                capital_distribution_summary="超大单净流入：+1.0亿",
                stock_news_summary="业绩超预期",
            )
        },
    )


@pytest.mark.unit
def test_parse_weekly_analysis_returns_structured_fields() -> None:
    gateway = _mock_gateway([_load_fixture("weekly_parsed_ok.json")])
    parsed = parse_weekly_analysis(gateway, "周线分析 markdown", session_id="s1", step=1)
    assert isinstance(parsed, WeeklyAnalysisParsed)
    assert parsed.support == 380.0
    assert parsed.resistance == 420.0
    assert parsed.short_term_trend == "bullish"


@pytest.mark.unit
def test_synthesize_pre_market_report_returns_valid_pydantic() -> None:
    gateway = _mock_gateway([_load_fixture("synthesis_ok.json")])
    working = _sample_working()
    context = PreMarketReportContext(working=working, code="00700.HK")
    synthesis = synthesize_pre_market_report(gateway, context, session_id="s1", step=2)
    assert isinstance(synthesis, PreMarketSynthesis)
    assert synthesis.result == "long"
    assert synthesis.suggestion == "buy"
    assert "腾讯控股" in synthesis.report


@pytest.mark.unit
def test_synthesize_raises_on_pydantic_validation_failure() -> None:
    gateway = _mock_gateway(
        [
            json.dumps(
                {
                    "result": "invalid_enum",
                    "confidence": "high",
                    "reason": "test",
                    "suggestion": "buy",
                    "report": "report body",
                    "summary": "summary",
                }
            )
        ]
    )
    working = _sample_working()
    context = PreMarketReportContext(working=working, code="00700.HK")
    with pytest.raises(LLMTaskError, match="pydantic validation failed"):
        synthesize_pre_market_report(gateway, context)


@pytest.mark.unit
def test_parse_weekly_analysis_raises_on_invalid_json() -> None:
    gateway = _mock_gateway(["not json at all"])
    with pytest.raises(LLMTaskError, match="not valid JSON"):
        parse_weekly_analysis(gateway, "markdown")


@pytest.mark.unit
def test_enrich_stock_with_llm_populates_workspace() -> None:
    gateway = _mock_gateway(
        [
            _load_fixture("weekly_parsed_ok.json"),
            _load_fixture("synthesis_ok.json"),
        ]
    )
    working = enrich_stock_with_llm(gateway, _sample_working(), "00700.HK", session_id="s1", step=3)
    ws = working.stocks["00700.HK"]
    assert ws.weekly_parsed is not None
    assert ws.weekly_parsed["support"] == 380.0
    assert ws.synthesis is not None
    assert ws.synthesis["result"] == "long"
    assert ws.status == "synthesizing"


@pytest.mark.unit
def test_build_report_uses_synthesis_when_present() -> None:
    working = _sample_working()
    working.stocks["00700.HK"].synthesis = json.loads(
        (FIXTURES / "synthesis_ok.json").read_text(encoding="utf-8")
    )
    content = build_report_content(working, "00700.HK")
    assert "综合预判" in content
    args = build_create_report_args(working, "00700.HK")
    assert args["result"] == "long"
    assert args["confidence"] == "high"
    assert args["bot_id"] == "bot-1"
