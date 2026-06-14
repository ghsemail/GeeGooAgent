"""Tool domain taxonomy — report workflow vs bot/reminder management vs market."""

from __future__ import annotations

from enum import StrEnum


class ToolDomain(StrEnum):
    """High-level purpose of a Tool (orthogonal to ToolCategory perceive/analyze/action)."""

    REPORT_WORKFLOW = "report_workflow"
    REPORT_QUERY = "report_query"
    BOT_MANAGER = "bot_manager"
    REMINDER_MANAGER = "reminder_manager"
    MARKET = "market"
    STRATEGY = "strategy"
    PROMPT_TEMPLATE = "prompt_template"
    META = "meta"


def _bot_crud(slug: str) -> frozenset[str]:
    return frozenset(
        {
            f"create_{slug}",
            f"update_{slug}",
            f"delete_{slug}",
            f"list_{slug}s",
            f"get_{slug}_log",
        }
    )


def _report_crud(slug: str) -> frozenset[str]:
    names = {
        f"update_{slug}_report",
        f"delete_{slug}_report",
        f"get_{slug}_reports",
    }
    if slug != "pre_market":
        names.add(f"create_{slug}_report")
    return frozenset(names)


BOT_MANAGER_TOOLS: frozenset[str] = frozenset().union(
    _bot_crud("dca_bot"),
    _bot_crud("grid_bot"),
    _bot_crud("smart_trade"),
    _bot_crud("hdg_bot"),
    {"switch_bot"},
)

REMINDER_MANAGER_TOOLS: frozenset[str] = frozenset().union(
    _bot_crud("dca_reminder"),
    _bot_crud("grid_reminder"),
    _bot_crud("smart_reminder"),
)

REPORT_QUERY_TOOLS: frozenset[str] = frozenset().union(
    _report_crud("pre_market"),
    _report_crud("intraday"),
    _report_crud("post_market"),
    {"get_stock_daily_reports", "list_today_reports"},
)

REPORT_WORKFLOW_TOOLS: frozenset[str] = frozenset(
    {
        "get_report_bot_codes",
        "create_pre_market_report",
        "save_local_report",
        "send_feishu_summary",
        "write_execution_log",
        "read_working_state",
        "recall_yesterday_summary",
        "get_bot_yesterday_attitude",
    }
)

MARKET_TOOLS: frozenset[str] = frozenset(
    {
        "check_trading_day",
        "search_code",
        "get_current_price",
        "get_ticker",
        "get_position",
        "get_broker",
        "get_capital_flow",
        "get_capital_distribution",
        "get_mcp_analysis",
        "get_single_prompt_template",
        "get_index_signals",
        "get_signal_combinations",
        "get_bot_log_by_type",
        "fetch_market_news",
        "fetch_stock_news",
        "recall",
    }
)

STRATEGY_TOOLS: frozenset[str] = frozenset(
    {
        "generate_grid_strategy",
        "generate_dca_strategy",
        "loopback_strategy",
    }
)

PROMPT_TEMPLATE_TOOLS: frozenset[str] = frozenset(
    {
        "create_competitor_prompt_template",
        "edit_competitor_prompt_template",
        "delete_competitor_prompt_template",
        "create_etf_prompt_template",
        "edit_etf_prompt_template",
        "delete_etf_prompt_template",
    }
)

# Interactive chat: market + bot/reminder ops + read reports. Excludes automated workflow writers.
CHAT_ON_DEMAND_TOOLS: list[str] = sorted(
    MARKET_TOOLS
    | STRATEGY_TOOLS
    | BOT_MANAGER_TOOLS
    | REMINDER_MANAGER_TOOLS
    | REPORT_QUERY_TOOLS
    - REPORT_WORKFLOW_TOOLS
)

DOMAIN_LABELS: dict[ToolDomain, str] = {
    ToolDomain.REPORT_WORKFLOW: "报告 Workflow（盘前/盘中/盘后自动化，勿用于查 Bot 列表）",
    ToolDomain.REPORT_QUERY: "报告查询（读盘前/盘中/盘后报告）",
    ToolDomain.BOT_MANAGER: "交易 Bot（DCA/GRID/SmartTrade/HDG）",
    ToolDomain.REMINDER_MANAGER: "提醒 Bot（DCA/GRID/Smart 提醒）",
    ToolDomain.MARKET: "行情与分析",
    ToolDomain.STRATEGY: "策略生成与回测",
    ToolDomain.PROMPT_TEMPLATE: "Prompt 模板",
    ToolDomain.META: "其他",
}

_TOOL_TO_DOMAIN: dict[str, ToolDomain] = {}


def _register(domain: ToolDomain, names: frozenset[str]) -> None:
    for name in names:
        _TOOL_TO_DOMAIN[name] = domain


_register(ToolDomain.REPORT_WORKFLOW, REPORT_WORKFLOW_TOOLS)
_register(ToolDomain.REPORT_QUERY, REPORT_QUERY_TOOLS)
_register(ToolDomain.BOT_MANAGER, BOT_MANAGER_TOOLS)
_register(ToolDomain.REMINDER_MANAGER, REMINDER_MANAGER_TOOLS)
_register(ToolDomain.MARKET, MARKET_TOOLS)
_register(ToolDomain.STRATEGY, STRATEGY_TOOLS)
_register(ToolDomain.PROMPT_TEMPLATE, PROMPT_TEMPLATE_TOOLS)


def tool_domain(name: str) -> ToolDomain:
    return _TOOL_TO_DOMAIN.get(name, ToolDomain.META)


def group_tool_names(names: list[str]) -> dict[ToolDomain, list[str]]:
    grouped: dict[ToolDomain, list[str]] = {domain: [] for domain in ToolDomain}
    for name in sorted(names):
        grouped[tool_domain(name)].append(name)
    return {domain: items for domain, items in grouped.items() if items}


CHAT_TOOL_ROUTING_RULES = """
Tool 路由（必须遵守）：
- 用户问「有哪些 / 列出 / 查询」**交易机器人** → list_dca_bots / list_grid_bots / list_smart_trades / list_hdg_bots
- 用户问「有哪些 / 列出」**提醒机器人**（含 GRID 网格提醒、DCA 提醒、Smart 提醒）
  → list_dca_reminders / list_grid_reminders / list_smart_reminders
- 用户问「今天的盘前/盘中/盘后报告」「某股某天的报告」→ get_stock_daily_reports / get_*_reports / list_today_reports
- **禁止**用 get_report_bot_codes 回答「有哪些机器人」——它仅用于盘前/盘后 Workflow，返回的是「开了态度监控、待写报告的标的」，不是 Reminder/Bot 全量列表
- 创建/修改 Bot 前先 search_code 确认标的，并向用户确认配置后再调用 create_* 
"""


def format_tools_listing(names: list[str], descriptions: dict[str, str]) -> str:
    lines: list[str] = []
    for domain, tool_names in group_tool_names(names).items():
        lines.append(f"[{DOMAIN_LABELS[domain]}]")
        for name in tool_names:
            desc = descriptions.get(name, "")
            short = desc[:72] + ("…" if len(desc) > 72 else "")
            lines.append(f"  - {name}: {short}")
        lines.append("")
    return "\n".join(lines).rstrip()
