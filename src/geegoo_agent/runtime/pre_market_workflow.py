"""Pre-market workflow step definitions."""

from __future__ import annotations

from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.runtime.pre_market_constants import (
    PRE_MARKET_INDEX_ENTRIES,
    PRE_MARKET_NEWS_MARKETS,
    TRADING_DAY_CHECK_CODE,
)
from geegoo_agent.runtime.pre_market_report import build_create_report_args, build_report_content
from geegoo_agent.runtime.workflow import WorkflowStep
from geegoo_agent.tools.analyze import INDEX_PROMPT_ID


def _stock_workspace(working: PreMarketWorking):
    code = working.current_stock_code
    if not code:
        raise ValueError("current_stock_code is not set")
    return working.stocks[code]


def _index_step(name: str, code: str) -> WorkflowStep:
    return WorkflowStep(
        f"index_{code}",
        "get_mcp_analysis",
        {
            "name": name,
            "code": code,
            "prompt_id": INDEX_PROMPT_ID,
            "period": "hourly",
            "language": "cn",
        },
    )


def _news_step(market: str) -> WorkflowStep:
    return WorkflowStep(
        f"market_news_{market.lower()}",
        "fetch_market_news",
        {"market": market, "limit": 8},
    )


PRE_MARKET_PHASE_A_STEPS: list[WorkflowStep] = [
    WorkflowStep("check_trading_day", "check_trading_day", {"code": TRADING_DAY_CHECK_CODE}),
    WorkflowStep("get_report_bot_codes", "get_report_bot_codes", {}),
    *[_index_step(name, code) for name, code in PRE_MARKET_INDEX_ENTRIES],
    *[_news_step(market) for market in PRE_MARKET_NEWS_MARKETS],
    WorkflowStep(
        "phase_a_complete",
        "write_execution_log",
        lambda w: {
            "step": "phase_a_complete",
            "message": (
                f"indices_done={w.market_context.indices_done} "
                f"market_news_done={w.market_context.market_news_done}"
            ),
            "status": "ok",
        },
    ),
]

PRE_MARKET_PER_STOCK_STEPS: list[WorkflowStep] = [
    WorkflowStep(
        "list_today_reports",
        "list_today_reports",
        lambda w: {"code": w.current_stock_code or ""},
    ),
    WorkflowStep(
        "stock_news",
        "fetch_stock_news",
        lambda w: {
            "code": w.current_stock_code or "",
            "stock_name": _stock_workspace(w).stock_name,
            "limit": 5,
        },
    ),
    WorkflowStep(
        "capital_flow",
        "get_capital_flow",
        lambda w: {"code": w.current_stock_code or "", "period": "DAY"},
    ),
    WorkflowStep(
        "capital_distribution",
        "get_capital_distribution",
        lambda w: {"code": w.current_stock_code or ""},
    ),
    WorkflowStep(
        "weekly_analysis",
        "get_mcp_analysis",
        lambda w: {
            "name": _stock_workspace(w).stock_name,
            "code": w.current_stock_code or "",
            "prompt_id": INDEX_PROMPT_ID,
            "period": "weekly",
            "language": "cn",
        },
    ),
    WorkflowStep(
        "bot_attitude",
        "get_bot_yesterday_attitude",
        lambda w: {
            "bot_id": _stock_workspace(w).bot_id,
            "code": w.current_stock_code or "",
            "language": "cn",
        },
    ),
    WorkflowStep(
        "save_local_report",
        "save_local_report",
        lambda w: {
            "code": w.current_stock_code or "",
            "content": build_report_content(w, w.current_stock_code or ""),
            "report_type": "premarket",
        },
    ),
    WorkflowStep(
        "create_pre_market_report",
        "create_pre_market_report",
        lambda w: build_create_report_args(w, w.current_stock_code or ""),
    ),
    WorkflowStep(
        "stock_complete",
        "write_execution_log",
        lambda w: {
            "step": f"stock_complete:{w.current_stock_code}",
            "message": f"status={_stock_workspace(w).status}",
            "status": "ok",
        },
    ),
]
