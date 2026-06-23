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
from geegoo_agent.tools.mappings import is_a_share
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
你是盘前报告合成助手。根据提供的多维度数据（市场概况、新闻、资金流向、资金分布、周线技术、Bot态度），生成结构化盘前研判报告。仅输出 JSON，不要其他文字。

=== 分析步骤（必须按顺序分析） ===

1. **市场概况分析**：根据各国指数走势判断整体市场情绪（bullish/bearish/mixed），分析美、A、港三大市场联动性与各自驱动力。必须输出具体市场判断，不能用"暂无"。

2. **新闻事件解读**：从市场新闻和个股新闻中提取 3-5 条最有影响力的消息，用自然语言概括要点，分析对标的的可能影响。禁止输出原始 JSON 数据结构。

3. **资金流向与分布分析**（仅港股/美股标的）：结合资金流向与资金分布判断资金态度。A 股（代码以 .SH/.SZ 结尾）无此数据，报告中不写「资金流向与分布」章节，综合判断也不引用该维度。

4. **周线技术简述**：从周线分析中提取关键支撑/阻力位、均线状态、趋势判断。

5. **Bot 盘前态度**：结合 Bot/Reminder 上一交易日态度，判断机器视角的倾向。

6. **综合判断**：综合以上维度，给出多空判断和置信度。
  - 判定依据必须包含具体参数引用（如"指数偏正面：道指+1.2%、纳指+0.8%"）
  - 置信度判断依据：信号一致性高（4+ 维度同向）→ high；3 维度同向 → medium；信号冲突 → low
  - 支撑/阻力从周线技术分析中提取

JSON 字段：
- result: long/short/neutral
- confidence: high/medium/low
- reason: 详细判定依据（≥80字，含具体参数和数据点）
- suggestion: buy/sell/hold
- report: 完整 Markdown 报告（严格按照九章结构，每章必须有实质性内容，不能写"暂无"）
- summary: 150-250字摘要，包含市场概况、关键风险、操作建议
- support, resistance: 数字或 null（从周线分析提取）
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

    # === 1. 市场概况（指数分析，每段落最多 500 字） ===
    market_parts = ["## 市场概况（指数分析）"]
    for label, idx_code in [
        ("道琼斯", "^DJI.US"),
        ("纳斯达克", "^IXIC.US"),
        ("上证指数", "000001.SH"),
        ("深证成指", "399001.SZ"),
        ("恒生指数", "800000.HK"),
    ]:
        analysis = mc.index_analysis_refs.get(idx_code, "暂无数据")
        market_parts.append(f"### {label} ({idx_code})\n{analysis[:500]}")

    # === 2. 市场新闻（做摘要预处理，每市场最多 600 字） ===
    market_parts.append("\n## 市场新闻")
    for mk, news_raw in mc.market_news.items():
        # 清理原始 JSON 格式的新闻数据
        cleaned = _clean_news_text(news_raw)
        market_parts.append(f"### {mk}市场新闻\n{cleaned[:600]}")

    # === 3. 个股数据 ===
    a_share = is_a_share(code)
    stock_parts = [
        f"## 个股信息",
        f"- 标的: {ws.stock_name} ({code})",
        f"- Bot ID: {ws.bot_id}",
    ]
    if a_share:
        stock_parts.append(
            "- 注意：A 股标的，资金流向与资金分布不可用，报告中勿写该章节。"
        )

    # Bot 态度
    attitude = ws.attitude or "neutral"
    stock_parts.append(f"\n## Bot 盘前态度\n{attitude}")

    # 周线分析
    weekly = ws.weekly_analysis_ref or "暂无"
    stock_parts.append(f"\n## 周线技术分析\n{weekly[:800]}")

    if context.weekly_parsed is not None:
        wp = context.weekly_parsed
        stock_parts.append(
            f"\n### 周线关键位\n"
            f"- 短期支撑: {wp.support_short}, 短期阻力: {wp.resistance_short}\n"
            f"- 中期支撑: {wp.support_mid}, 中期阻力: {wp.resistance_mid}\n"
            f"- 短期趋势: {wp.short_term_trend}, 中期趋势: {wp.mid_term_trend}, 长期趋势: {wp.long_term_trend}"
        )

    # 资金流向与分布（A 股跳过）
    if not a_share:
        flow = ws.capital_flow_summary or "暂无"
        stock_parts.append(f"\n## 资金流向\n{flow}")
        dist = ws.capital_distribution_summary or "暂无"
        stock_parts.append(f"\n## 资金分布\n{dist}")

    # 个股新闻
    stock_news = _clean_news_text(ws.stock_news_summary or "暂无")
    stock_parts.append(f"\n## 个股新闻\n{stock_news[:600]}")

    return "\n\n".join(market_parts) + "\n\n" + "\n\n".join(stock_parts)


def _clean_news_text(raw: str) -> str:
    """清洗新闻原始文本：去除 JSON 结构标记，提取可读摘要。"""
    import json as _json

    text = (raw or "").strip()
    if not text or text in ("暂无", "暂无数据"):
        return "暂无相关新闻"

    # 尝试解析 JSON，提取标题/内容
    try:
        data = _json.loads(text)
        if isinstance(data, list):
            items = []
            for item in data[:8]:
                if isinstance(item, dict):
                    title = item.get("title") or item.get("Title") or ""
                    summary = item.get("summary") or item.get("Summary") or item.get("content", "")[:100] or ""
                    line = f"- {title}"
                    if summary and summary != title:
                        line += f"：{summary}"
                    items.append(line)
                else:
                    items.append(f"- {str(item)[:150]}")
            return "\n".join(items) if items else "暂无相关新闻"
        elif isinstance(data, dict):
            items = data.get("items") or data.get("data") or data.get("news") or []
            if isinstance(items, list) and items:
                return _clean_news_text(_json.dumps(items, ensure_ascii=False))
            return str(data)[:600]
    except (_json.JSONDecodeError, ValueError):
        pass

    # 非 JSON：保持原样但截断
    return text[:800]


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
