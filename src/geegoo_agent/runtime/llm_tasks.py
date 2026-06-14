"""LLM structured tasks for pre-market workflow."""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel, ValidationError

from geegoo_agent.exceptions import LLMTaskError
from geegoo_agent.llm.gateway import ModelGateway
from geegoo_agent.llm.types import Message
from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.tools.schemas import ConfidenceType, ResultType, SuggestionType

_JSON_FENCE_RE = re.compile(r"```(?:json)?\s*([\s\S]*?)```", re.IGNORECASE)

_PARSE_WEEKLY_SYSTEM = """\
你是证券技术分析助手。从周线分析 Markdown 中提取结构化字段，仅输出 JSON，不要其他文字。
JSON 字段：
- support_short, support_mid, support_long: 支撑位（数字或 null）
- resistance_short, resistance_mid, resistance_long: 阻力位（数字或 null）
- short_term_trend, mid_term_trend, long_term_trend: bullish/bearish/neutral
- trading_suggestion: 操作建议文本
- support, resistance: 主要支撑/阻力（数字或 null，用于 API）
"""

_SYNTHESIZE_SYSTEM = """\
你是盘前报告合成助手。根据提供的指数、新闻、个股数据生成盘前研判，仅输出 JSON。
JSON 字段：
- result: long/short/neutral
- confidence: high/medium/low
- reason: 非空判定依据
- suggestion: buy/sell/hold
- report: 完整 Markdown 报告（九章结构摘要即可）
- summary: 200 字以内摘要
- support, resistance: 数字或 null
"""


class WeeklyAnalysisParsed(BaseModel):
    support_short: float | None = None
    support_mid: float | None = None
    support_long: float | None = None
    resistance_short: float | None = None
    resistance_mid: float | None = None
    resistance_long: float | None = None
    short_term_trend: str = "neutral"
    mid_term_trend: str = "neutral"
    long_term_trend: str = "neutral"
    trading_suggestion: str = ""
    support: float | None = None
    resistance: float | None = None


class PreMarketSynthesis(BaseModel):
    result: ResultType
    confidence: ConfidenceType
    reason: str
    suggestion: SuggestionType
    report: str
    summary: str
    support: float | None = None
    resistance: float | None = None


@dataclass
class PreMarketReportContext:
    working: PreMarketWorking
    code: str
    weekly_parsed: WeeklyAnalysisParsed | None = None


def _extract_json(text: str) -> dict[str, Any]:
    stripped = text.strip()
    match = _JSON_FENCE_RE.search(stripped)
    payload = match.group(1).strip() if match else stripped
    try:
        data = json.loads(payload)
    except json.JSONDecodeError as exc:
        raise LLMTaskError(f"LLM response is not valid JSON: {exc}") from exc
    if not isinstance(data, dict):
        raise LLMTaskError("LLM response JSON must be an object")
    return data


def _validate_model(model_cls: type[BaseModel], data: dict[str, Any], task: str) -> BaseModel:
    try:
        return model_cls.model_validate(data)
    except ValidationError as exc:
        raise LLMTaskError(f"{task} pydantic validation failed: {exc}") from exc


def _chat_json(
    gateway: ModelGateway,
    system: str,
    user: str,
    *,
    session_id: str,
    step: int,
    task: str,
    model_cls: type[BaseModel],
) -> BaseModel:
    response = gateway.chat(
        [Message(role="system", content=system), Message(role="user", content=user)],
        [],
        session_id=session_id,
        step=step,
    )
    content = response.content or ""
    data = _extract_json(content)
    return _validate_model(model_cls, data, task)


def parse_weekly_analysis(
    gateway: ModelGateway,
    markdown: str,
    *,
    session_id: str = "default",
    step: int = 0,
) -> WeeklyAnalysisParsed:
    """Parse weekly MCP analysis markdown into structured fields."""
    result = _chat_json(
        gateway,
        _PARSE_WEEKLY_SYSTEM,
        markdown or "（无周线分析内容）",
        session_id=session_id,
        step=step,
        task="parse_weekly_analysis",
        model_cls=WeeklyAnalysisParsed,
    )
    assert isinstance(result, WeeklyAnalysisParsed)
    return result


def _build_synthesis_prompt(context: PreMarketReportContext) -> str:
    working = context.working
    code = context.code
    ws = working.stocks[code]
    mc = working.market_context
    parts = [
        f"标的: {ws.stock_name} ({code})",
        f"昨日 Bot 态度: {ws.attitude or 'neutral'}",
        f"周线分析原文:\n{ws.weekly_analysis_ref or '暂无'}",
        f"资金流向: {ws.capital_flow_summary or '暂无'}",
        f"资金分布:\n{ws.capital_distribution_summary or '暂无'}",
        f"个股新闻:\n{ws.stock_news_summary or '暂无'}",
        "指数分析:",
    ]
    for idx_code, analysis in mc.index_analysis_refs.items():
        parts.append(f"- {idx_code}: {analysis[:300]}")
    for market, news in mc.market_news.items():
        parts.append(f"市场新闻 {market}: {news[:300]}")
    if context.weekly_parsed is not None:
        parts.append(f"周线结构化: {context.weekly_parsed.model_dump_json()}")
    return "\n\n".join(parts)


def synthesize_pre_market_report(
    gateway: ModelGateway,
    context: PreMarketReportContext,
    *,
    session_id: str = "default",
    step: int = 0,
) -> PreMarketSynthesis:
    """Synthesize pre-market report fields from working memory context."""
    prompt = _build_synthesis_prompt(context)
    result = _chat_json(
        gateway,
        _SYNTHESIZE_SYSTEM,
        prompt,
        session_id=session_id,
        step=step,
        task="synthesize_pre_market_report",
        model_cls=PreMarketSynthesis,
    )
    assert isinstance(result, PreMarketSynthesis)
    return result


def enrich_stock_with_llm(
    gateway: ModelGateway,
    working: PreMarketWorking,
    code: str,
    *,
    session_id: str = "default",
    step: int = 0,
) -> PreMarketWorking:
    """Run parse + synthesize for one stock and store results on workspace."""
    if code not in working.stocks:
        raise ValueError(f"stock not in working memory: {code}")

    ws = working.stocks[code]
    weekly_md = ws.weekly_analysis_ref or ""
    weekly_parsed = parse_weekly_analysis(
        gateway,
        weekly_md,
        session_id=session_id,
        step=step,
    )

    updated = working.model_copy(deep=True)
    updated.stocks[code].weekly_parsed = weekly_parsed.model_dump()
    updated.stocks[code].status = "synthesizing"

    context = PreMarketReportContext(
        working=updated,
        code=code,
        weekly_parsed=weekly_parsed,
    )
    synthesis = synthesize_pre_market_report(
        gateway,
        context,
        session_id=session_id,
        step=step,
    )
    updated.stocks[code].synthesis = synthesis.model_dump()
    return updated
