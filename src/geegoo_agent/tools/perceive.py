"""Perception tools."""

from __future__ import annotations

from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ClientError, ConfigError
from geegoo_agent.runtime.pre_market_constants import DRY_RUN_SAMPLE_BOTS
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class CheckTradingDayParams(BaseModel):
    code: str = Field(default="00700.HK", description="Stock code for market inference")


class GetCurrentPriceParams(BaseModel):
    code: str = Field(description="Ticker e.g. 00700.HK, 600519.SH")


class GetReportBotCodesParams(BaseModel):
    pass


class CheckTradingDayTool(BaseTool):
    name = "check_trading_day"
    description = "Check if today is a trading day for the market of the given code."
    category = ToolCategory.PERCEPTION
    input_model = CheckTradingDayParams

    def run(self, params: CheckTradingDayParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary="dry-run: skipped check_trading_day",
                data={"is_trading_day": True},
            )
        data = ctx.market_client.check_trading_day(ctx.mcp_token, params.code)
        return ToolResult(
            status="ok",
            summary=(
                f"Trading day check: {data.code} on {data.date} "
                f"market={data.market} is_trading_day={data.is_trading_day}"
            ),
            data=data.model_dump(),
        )


class GetCurrentPriceTool(BaseTool):
    name = "get_current_price"
    description = (
        "Get latest price via geegoo mcp getCurrentPrice; falls back to getTicker on failure."
    )
    category = ToolCategory.PERCEPTION
    input_model = GetCurrentPriceParams

    def run(self, params: GetCurrentPriceParams, ctx: ToolContext) -> ToolResult:
        code = str(params.code).strip()
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped get_current_price for {code}",
                data={"code": code, "price": None},
            )
        if ctx.geegoo_bot_client is None:
            raise ConfigError("geegoo_bot_client not configured")

        primary_error: str | None = None
        try:
            payload = ctx.geegoo_bot_client.post_direct(
                "/getCurrentPrice",
                {"mcp_token": ctx.mcp_token, "code": code},
            )
            if isinstance(payload, dict) and payload.get("price") is not None:
                price = payload["price"]
                return ToolResult(
                    status="ok",
                    summary=f"{code} price={price}",
                    data={"code": code, "price": price, "source": "5700"},
                )
        except ClientError as exc:
            primary_error = str(exc)

        fallback = self._price_from_ticker(ctx, code)
        if fallback is not None:
            price, tick_time = fallback
            note = f" (5700 failed: {primary_error})" if primary_error else ""
            return ToolResult(
                status="ok",
                summary=f"{code} price={price} via get_ticker{note}",
                data={
                    "code": code,
                    "price": price,
                    "source": "5700/get_ticker",
                    "time": tick_time,
                    "primary_error": primary_error,
                },
            )

        msg = primary_error or "no price from getCurrentPrice or getTicker"
        return ToolResult(status="error", summary=f"get_current_price failed: {msg}", exit_code=1)

    @staticmethod
    def _price_from_ticker(ctx: ToolContext, code: str) -> tuple[float, str | None] | None:
        try:
            payload = ctx.market_client.post(
                "/getTicker",
                {"mcp_token": ctx.mcp_token, "code": code, "num": 5},
            )
        except ClientError:
            return None
        items = payload.get("data") if isinstance(payload, dict) else None
        if not isinstance(items, list):
            items = payload.get("items") if isinstance(payload, dict) else None
        if not items:
            return None
        last = items[-1] if isinstance(items[-1], dict) else None
        if not last or last.get("price") is None:
            return None
        return float(last["price"]), last.get("time")


class GetReportBotCodesTool(BaseTool):
    name = "get_report_bot_codes"
    description = (
        "Report workflow ONLY: list stocks/bots with attitude.switch=true for pre/post-market "
        "reports (code, bot_id, bot_name, bot_type). "
        "Do NOT use for 'list my bots/reminders' — use list_grid_reminders / list_grid_bots instead."
    )
    category = ToolCategory.PERCEPTION
    input_model = GetReportBotCodesParams

    def run(self, params: GetReportBotCodesParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: {len(DRY_RUN_SAMPLE_BOTS)} sample bot(s)",
                data={"bots": list(DRY_RUN_SAMPLE_BOTS)},
            )
        bots = ctx.market_client.get_report_bot_codes(ctx.mcp_token)
        payload = [b.model_dump() for b in bots]
        return ToolResult(
            status="ok",
            summary=f"Found {len(bots)} bot(s) under monitoring.",
            data={"bots": payload},
        )
