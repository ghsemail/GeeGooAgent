"""TradingBot API tool catalog — grouped by L2 category and source doc."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Literal

from geegoo_agent.tools.types import ToolCategory

ClientName = Literal["market", "geegoo_bot"]
DocName = Literal[
    "common",
    "trading",
    "market",
    "analyst",
    "strategy",
    "loopback",
    "dca_bot",
    "grid_bot",
    "smart_trade",
    "hdg_bot",
    "dca_reminder",
    "grid_reminder",
    "smart_reminder",
    "local",
]


@dataclass(frozen=True)
class FieldSpec:
    name: str
    type_name: Literal["str", "int", "float", "bool", "dict", "list"] = "str"
    required: bool = False
    default: Any = None
    description: str = ""


ResponseMode = Literal["mcp", "direct"]


@dataclass(frozen=True)
class HttpToolSpec:
    name: str
    description: str
    category: ToolCategory
    client: ClientName
    path: str
    doc: DocName
    requires_mcp_token: bool = True
    fields: tuple[FieldSpec, ...] = ()
    merge_payload: bool = False
    response_mode: ResponseMode = "mcp"


# Names implemented as bespoke tools (see perceive/analyze/act_*.py).
BESPOKE_TOOL_NAMES: frozenset[str] = frozenset(
    {
        "check_trading_day",
        "get_current_price",
        "get_report_bot_codes",
        "fetch_market_news",
        "fetch_stock_news",
        "get_mcp_analysis",
        "get_stock_daily_reports",
        "list_today_reports",
        "get_capital_flow",
        "get_capital_distribution",
        "get_bot_yesterday_attitude",
        "recall",
        "recall_yesterday_summary",
        "read_working_state",
        "create_pre_market_report",
        "save_local_report",
        "send_feishu_summary",
        "write_execution_log",
    }
)


def _payload_field(description: str = "API body fields (merged into request)") -> FieldSpec:
    return FieldSpec("payload", "dict", required=False, default={}, description=description)


def _bot_crud(
    slug: str,
    label: str,
    create_path: str,
    update_path: str,
    delete_path: str,
    list_path: str,
    log_path: str,
    doc: DocName,
) -> list[HttpToolSpec]:
    return [
        HttpToolSpec(
            f"create_{slug}",
            f"Create {label}.",
            ToolCategory.ACTION,
            "geegoo_bot",
            create_path,
            doc,
            fields=(_payload_field(),),
            merge_payload=True,
        ),
        HttpToolSpec(
            f"update_{slug}",
            f"Update {label} by bot_id.",
            ToolCategory.ACTION,
            "geegoo_bot",
            update_path,
            doc,
            fields=(
                FieldSpec("bot_id", required=True, description="Bot ID"),
                _payload_field(),
            ),
            merge_payload=True,
        ),
        HttpToolSpec(
            f"delete_{slug}",
            f"Delete {label} by bot_id.",
            ToolCategory.ACTION,
            "geegoo_bot",
            delete_path,
            doc,
            fields=(FieldSpec("bot_id", required=True, description="Bot ID"),),
        ),
        HttpToolSpec(
            f"list_{slug}s",
            f"List all user {label}s ({list_path}). For bot/reminder inventory — NOT report workflow.",
            ToolCategory.ACTION,
            "geegoo_bot",
            list_path,
            doc,
        ),
        HttpToolSpec(
            f"get_{slug}_log",
            f"Get run log for {label}.",
            ToolCategory.ANALYSIS,
            "geegoo_bot",
            log_path,
            doc,
            fields=(FieldSpec("bot_id", required=True, description="Bot ID"),),
        ),
    ]


def _report_crud(
    slug: str,
    label: str,
    create_path: str,
    update_path: str,
    delete_path: str,
    list_path: str,
) -> list[HttpToolSpec]:
    specs: list[HttpToolSpec] = []
    if slug != "pre_market":
        specs.append(
            HttpToolSpec(
                f"create_{slug}_report",
                f"Create {label}.",
                ToolCategory.ACTION,
                "geegoo_bot",
                create_path,
                "geegoo_bot",
                fields=(_payload_field(),),
                merge_payload=True,
            )
        )
    specs.extend(
        [
            HttpToolSpec(
                f"update_{slug}_report",
                f"Update {label} by report_id.",
                ToolCategory.ACTION,
                "geegoo_bot",
                update_path,
                "geegoo_bot",
                fields=(
                    FieldSpec("report_id", required=True, description="Report ID"),
                    _payload_field("Fields to update"),
                ),
                merge_payload=True,
            ),
            HttpToolSpec(
                f"delete_{slug}_report",
                f"Delete {label} by report_id.",
                ToolCategory.ACTION,
                "geegoo_bot",
                delete_path,
                "geegoo_bot",
                fields=(FieldSpec("report_id", required=True, description="Report ID"),),
            ),
            HttpToolSpec(
                f"get_{slug}_reports",
                f"Query stored {label} documents (read-only). Not the same as list_*_bots/reminders.",
                ToolCategory.ANALYSIS,
                "geegoo_bot",
                list_path,
                "geegoo_bot",
                fields=(
                    FieldSpec("code", description="Stock code filter"),
                    FieldSpec("report_date", description="YYYY-MM-DD filter"),
                ),
            ),
        ]
    )
    return specs


# --- Perception (MCP_API_Common.md + Trading) ---
PERCEPTION_TOOLS: list[HttpToolSpec] = [
    HttpToolSpec(
        "search_code",
        "Search stock by code or name (Signal /searchCode). No mcp_token.",
        ToolCategory.PERCEPTION,
        "geegoo_bot",
        "/searchCode",
        "common",
        requires_mcp_token=False,
        response_mode="direct",
        fields=(
            FieldSpec("regex", required=True, description="Code or name keyword"),
            FieldSpec("market", "list", description='Market filter e.g. ["HK","US"]'),
        ),
    ),
    HttpToolSpec(
        "get_position",
        "Get account position for a symbol (cost, qty, P/L). Requires mcp_token.",
        ToolCategory.PERCEPTION,
        "geegoo_bot",
        "/getPosition",
        "common",
        fields=(FieldSpec("code", required=True, description="Ticker code"),),
    ),
    HttpToolSpec(
        "get_current_price",
        "Get latest price via TradingServer (Futu HK/CN, USData US). Requires mcp_token.",
        ToolCategory.PERCEPTION,
        "geegoo_bot",
        "/getCurrentPrice",
        "common",
        response_mode="direct",
        fields=(FieldSpec("code", required=True, description="Ticker e.g. 00700.HK"),),
    ),
    HttpToolSpec(
        "get_ticker",
        "Get real-time ticker for a symbol (geegoo mcp).",
        ToolCategory.PERCEPTION,
        "geegoo_bot",
        "/getTicker",
        "trading",
        fields=(FieldSpec("code", required=True, description="Ticker code"),),
    ),
    HttpToolSpec(
        "get_broker",
        "Get broker distribution for a symbol (geegoo mcp).",
        ToolCategory.PERCEPTION,
        "geegoo_bot",
        "/getBroker",
        "trading",
        fields=(FieldSpec("code", required=True, description="Ticker code"),),
    ),
]

# --- Analysis (signals, templates, logs, strategy) ---
ANALYSIS_TOOLS: list[HttpToolSpec] = [
    HttpToolSpec(
        "get_index_signals",
        "List index signals from Admin (body may be {}). No mcp_token.",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/getIndexSignalForSkill",
        "common",
        requires_mcp_token=False,
        response_mode="direct",
    ),
    HttpToolSpec(
        "get_signal_combinations",
        "List combined signals from Admin (body may be {}). No mcp_token.",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/getSignalCombinationForSkill",
        "common",
        requires_mcp_token=False,
        response_mode="direct",
    ),
    HttpToolSpec(
        "get_single_prompt_template",
        "List prompt templates (type=tech|index|fundamental).",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/getSinglePromptTemplate",
        "analyst",
        response_mode="direct",
        fields=(
            FieldSpec("type", required=True, description="tech, index, or fundamental"),
            FieldSpec("period", description="Optional period filter"),
        ),
    ),
    HttpToolSpec(
        "get_bot_log_by_type",
        "Query bot log by type and bot_id (DCA/GRID/Reminder).",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/getBotLogByType",
        "trading",
        fields=(
            FieldSpec("type", required=True, description="DCA, GRID, DCAReminder, etc."),
            FieldSpec("bot_id", required=True, description="Bot ID"),
        ),
    ),
    HttpToolSpec(
        "generate_grid_strategy",
        "Generate GRID strategy recommendation via AIServer.",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/generateGridStrategy",
        "strategy",
        fields=(
            FieldSpec("code", required=True),
            FieldSpec("name", required=True),
            FieldSpec("months_back", "int", description="History months, default 6"),
        ),
    ),
    HttpToolSpec(
        "generate_dca_strategy",
        "Generate DCA strategy with signal evaluation.",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/generateDCAStrategy",
        "strategy",
        fields=(
            FieldSpec("code", required=True),
            FieldSpec("name", required=True),
            FieldSpec("signal_id", required=True),
            FieldSpec("months_back", "int", description="History months"),
        ),
    ),
    HttpToolSpec(
        "loopback_strategy",
        "Backtest DCA or GRID strategy via Signal Server.",
        ToolCategory.ANALYSIS,
        "geegoo_bot",
        "/loopBackStrategy",
        "loopback",
        fields=(_payload_field("strategy_type, code, frequency, fund, grid_param, etc."),),
        merge_payload=True,
    ),
]

# --- Action: Market reports (pre/intraday/post) ---
MARKET_REPORT_TOOLS: list[HttpToolSpec] = []
MARKET_REPORT_TOOLS.extend(_report_crud(
    "pre_market",
    "pre-market report",
    "/createPreMarketReport",
    "/updatePreMarketReport",
    "/deletePreMarketReport",
    "/getPreMarketReports",
))
MARKET_REPORT_TOOLS.extend(_report_crud(
    "intraday",
    "intraday trade decision report",
    "/createIntradayTradeDecisionReport",
    "/updateIntradayTradeDecisionReport",
    "/deleteIntradayTradeDecisionReport",
    "/getIntradayTradeDecisionReports",
))
MARKET_REPORT_TOOLS.extend(_report_crud(
    "post_market",
    "post-market report",
    "/createPostMarketReport",
    "/updatePostMarketReport",
    "/deletePostMarketReport",
    "/getPostMarketReports",
))

# --- Action: Prompt template CRUD ---
PROMPT_TEMPLATE_TOOLS: list[HttpToolSpec] = [
    HttpToolSpec(
        "create_competitor_prompt_template",
        "Create user competitor analysis prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/createCompetitorPromptTemplate",
        "analyst",
        fields=(_payload_field(),),
        merge_payload=True,
    ),
    HttpToolSpec(
        "edit_competitor_prompt_template",
        "Edit competitor prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/editCompetitorPromptTemplate",
        "analyst",
        fields=(_payload_field("id, list, variable, etc."),),
        merge_payload=True,
    ),
    HttpToolSpec(
        "delete_competitor_prompt_template",
        "Delete competitor prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/deleteCompetitorPromptTemplate",
        "analyst",
        fields=(_payload_field("id"),),
        merge_payload=True,
    ),
    HttpToolSpec(
        "create_etf_prompt_template",
        "Create ETF analysis prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/createEtfPromptTemplate",
        "analyst",
        fields=(_payload_field(),),
        merge_payload=True,
    ),
    HttpToolSpec(
        "edit_etf_prompt_template",
        "Edit ETF prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/editEtfPromptTemplate",
        "analyst",
        fields=(_payload_field("id, list, variable, etc."),),
        merge_payload=True,
    ),
    HttpToolSpec(
        "delete_etf_prompt_template",
        "Delete ETF prompt template.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/deleteEtfPromptTemplate",
        "analyst",
        fields=(_payload_field("id"),),
        merge_payload=True,
    ),
]

# --- Action: Trading bots ---
BOT_TOOLS: list[HttpToolSpec] = []
BOT_TOOLS.extend(_bot_crud(
    "dca_bot",
    "DCA trading bot",
    "/createDCABot",
    "/updateDCABot",
    "/deleteDCABot",
    "/getAllDCABots",
    "/getDCABotLog",
    "dca_bot",
))
BOT_TOOLS.extend(_bot_crud(
    "grid_bot",
    "GRID trading bot",
    "/createGRIDBot",
    "/updateGRIDBot",
    "/deleteGRIDBot",
    "/getAllGRIDBots",
    "/getGRIDBotLog",
    "grid_bot",
))
BOT_TOOLS.extend(_bot_crud(
    "smart_trade",
    "SmartTrade bot",
    "/createSmartTrade",
    "/updateSmartTrade",
    "/deleteSmartTrade",
    "/getAllSmartTrades",
    "/getSmartTradeLog",
    "smart_trade",
))
BOT_TOOLS.extend(_bot_crud(
    "hdg_bot",
    "HDG hedging bot",
    "/createHDGBot",
    "/updateHDGBot",
    "/deleteHDGBot",
    "/getAllHDGBots",
    "/getHDGBotLog",
    "hdg_bot",
))

# --- Action: Reminder bots ---
REMINDER_TOOLS: list[HttpToolSpec] = []
REMINDER_TOOLS.extend(_bot_crud(
    "dca_reminder",
    "DCA reminder",
    "/createDCAReminder",
    "/updateDCAReminder",
    "/deleteDCAReminder",
    "/getAllDCAReminders",
    "/getDCAReminderLog",
    "dca_reminder",
))
REMINDER_TOOLS.extend(_bot_crud(
    "grid_reminder",
    "GRID reminder",
    "/createGRIDReminder",
    "/updateGRIDReminder",
    "/deleteGRIDReminder",
    "/getAllGRIDReminders",
    "/getGRIDReminderLog",
    "grid_reminder",
))
REMINDER_TOOLS.extend(_bot_crud(
    "smart_reminder",
    "Smart reminder",
    "/createSmartReminder",
    "/updateSmartReminder",
    "/deleteSmartReminder",
    "/getAllSmartReminders",
    "/getSmartReminderLog",
    "smart_reminder",
))

SWITCH_TOOLS: list[HttpToolSpec] = [
    HttpToolSpec(
        "switch_bot",
        "Enable or disable a bot/reminder.",
        ToolCategory.ACTION,
        "geegoo_bot",
        "/switchBot",
        "common",
        fields=(
            FieldSpec("bot_id", required=True),
            FieldSpec("bot_type", required=True, description="DCA, GRID, SmartReminder, etc."),
            FieldSpec("switch", "bool", description="true=on, false=off"),
        ),
    ),
]

HTTP_TOOL_CATALOG: list[HttpToolSpec] = [
    *PERCEPTION_TOOLS,
    *ANALYSIS_TOOLS,
    *MARKET_REPORT_TOOLS,
    *PROMPT_TEMPLATE_TOOLS,
    *BOT_TOOLS,
    *REMINDER_TOOLS,
    *SWITCH_TOOLS,
]

CATALOG_BY_CATEGORY: dict[ToolCategory, list[HttpToolSpec]] = {
    ToolCategory.PERCEPTION: PERCEPTION_TOOLS,
    ToolCategory.ANALYSIS: ANALYSIS_TOOLS,
    ToolCategory.ACTION: [
        *MARKET_REPORT_TOOLS,
        *PROMPT_TEMPLATE_TOOLS,
        *BOT_TOOLS,
        *REMINDER_TOOLS,
        *SWITCH_TOOLS,
    ],
}

CATALOG_BY_DOC: dict[DocName, list[HttpToolSpec]] = {}
for _spec in HTTP_TOOL_CATALOG:
    CATALOG_BY_DOC.setdefault(_spec.doc, []).append(_spec)


SPEC_BY_NAME: dict[str, HttpToolSpec] = {spec.name: spec for spec in HTTP_TOOL_CATALOG}


def catalog_http_specs(*, exclude_bespoke: bool = True) -> list[HttpToolSpec]:
    if not exclude_bespoke:
        return list(HTTP_TOOL_CATALOG)
    return [s for s in HTTP_TOOL_CATALOG if s.name not in BESPOKE_TOOL_NAMES]
