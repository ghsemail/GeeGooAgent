"""Working memory store and apply logic."""

from __future__ import annotations

from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.models import BotStock, PreMarketWorking, StockWorkspace
from geegoo_agent.runtime.pre_market_constants import PRE_MARKET_INDEX_CODES, PRE_MARKET_NEWS_MARKETS
from geegoo_agent.tools.types import ToolResult


class WorkingMemoryStore:
    def __init__(self, store: FileStateStore) -> None:
        self._store = store

    def _key(self, session_id: str) -> str:
        return f"working/{session_id}"

    def create(self, session_id: str, *, skill: str = "pre_market") -> PreMarketWorking:
        working = PreMarketWorking(session_id=session_id, skill=skill)
        self.save(working)
        return working

    def load(self, session_id: str) -> PreMarketWorking | None:
        data = self._store.load(self._key(session_id))
        if data is None:
            return None
        return PreMarketWorking.model_validate(data)

    def save(self, working: PreMarketWorking) -> None:
        self._store.save(self._key(working.session_id), working.model_dump())

    def apply(
        self,
        working: PreMarketWorking,
        tool_name: str,
        result: ToolResult,
    ) -> PreMarketWorking:
        updated = working.model_copy(deep=True)
        step_key = f"{tool_name}:{result.status}"
        if step_key not in updated.steps_completed:
            updated.steps_completed.append(step_key)

        if tool_name == "check_trading_day" and result.data is not None:
            updated.is_trading_day = bool(result.data.get("is_trading_day"))
            if updated.is_trading_day:
                updated.phase = "phase_a"
            else:
                updated.phase = "done"

        elif tool_name == "get_report_bot_codes" and result.data is not None:
            bots_raw = result.data.get("bots", [])
            updated.bot_codes = [BotStock.model_validate(b) for b in bots_raw]
            for bot in updated.bot_codes:
                if bot.code not in updated.stocks:
                    updated.stocks[bot.code] = StockWorkspace(
                        code=bot.code,
                        stock_name=bot.stock_name,
                        bot_id=bot.bot_id,
                        bot_name=bot.bot_name,
                        bot_type=bot.bot_type,
                    )
        elif tool_name == "get_mcp_analysis" and result.data is not None:
            code = str(result.data.get("code", ""))
            period = str(result.data.get("period", ""))
            analysis = str(result.data.get("analysis_result", ""))
            if code in PRE_MARKET_INDEX_CODES:
                updated.market_context.index_analysis_refs[code] = analysis[:2000]
                if code not in updated.market_context.index_codes_done:
                    updated.market_context.index_codes_done.append(code)
                if PRE_MARKET_INDEX_CODES <= set(updated.market_context.index_codes_done):
                    updated.market_context.indices_done = True
            elif code and code in updated.stocks and period == "weekly":
                updated.stocks[code].weekly_analysis_ref = analysis[:2000]

        elif tool_name == "fetch_stock_news" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks:
                updated.stocks[code].stock_news_summary = str(result.data.get("text", ""))[:2000]

        elif tool_name == "get_capital_flow" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks:
                if result.status == "skipped":
                    updated.stocks[code].capital_flow_summary = str(
                        result.data.get("skip_reason") or result.summary
                    )[:2000]
                else:
                    latest = result.data.get("latest") or {}
                    main_flow = latest.get("main_in_flow", "n/a")
                    updated.stocks[code].capital_flow_summary = f"main_in_flow={main_flow}"

        elif tool_name == "get_capital_distribution" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks:
                updated.stocks[code].capital_distribution_summary = str(
                    result.data.get("formatted", "") or result.summary
                )[:2000]

        elif tool_name == "get_bot_yesterday_attitude" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks:
                updated.stocks[code].attitude = str(result.data.get("attitude", "neutral"))

        elif tool_name == "list_today_reports" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks and result.data.get("already_reported"):
                updated.stocks[code].status = "skipped"

        elif tool_name == "save_local_report" and result.data is not None:
            code = str(result.data.get("code", ""))
            path = result.data.get("path")
            if code and code in updated.stocks and path:
                updated.stocks[code].report_ref = str(path)

        elif tool_name == "create_pre_market_report" and result.data is not None:
            code = str(result.data.get("code", ""))
            if code and code in updated.stocks:
                updated.stocks[code].status = "reported"
                report_id = result.data.get("report_id")
                if report_id:
                    updated.stocks[code].report_id = str(report_id)

        elif tool_name == "fetch_market_news" and result.data is not None:
            market = str(result.data.get("market", ""))
            if market:
                updated.market_context.market_news[market] = str(result.data.get("text", ""))[:2000]
            if set(updated.market_context.market_news.keys()) >= set(PRE_MARKET_NEWS_MARKETS):
                updated.market_context.market_news_done = True

        elif tool_name == "write_execution_log" and result.data is not None:
            path = result.data.get("path")
            if path:
                updated.artifacts["execution_log"] = str(path)

        self.save(updated)
        return updated

    def summary(self, working: PreMarketWorking) -> str:
        reported = sum(1 for s in working.stocks.values() if s.status == "reported")
        total = len(working.stocks)
        pending = [
            f"{code}({ws.status})"
            for code, ws in working.stocks.items()
            if ws.status not in {"reported", "skipped", "failed"}
        ]
        parts = [
            f"phase={working.phase}",
            f"trading_day={working.is_trading_day}",
            f"bots={len(working.bot_codes)}",
            f"reported={reported}/{total}",
        ]
        if pending:
            parts.append(f"pending: {', '.join(pending)}")
        return " | ".join(parts)
