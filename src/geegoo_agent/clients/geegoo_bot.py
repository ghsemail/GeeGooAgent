"""GeeGoo MCP API client (port 5700)."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field

from geegoo_agent.clients.base import BaseClient
from geegoo_agent.exceptions import ClientError

McpAnalysisPeriod = Literal[
    "no_period",
    "minutes",
    "hourly",
    "daily",
    "weekly",
    "monthly",
    "quarterly",
    "yearly",
    "longterm",
]

CapitalFlowPeriod = Literal["INTRADAY", "DAY", "WEEK", "MONTH", "YEAR"]


class McpAnalysisData(BaseModel):
    analysis_result: str = ""
    model: str | None = None
    create_date: str | None = None
    extra: dict[str, Any] = Field(default_factory=dict)

    @classmethod
    def from_api(cls, data: dict[str, Any]) -> McpAnalysisData:
        known = {k: data[k] for k in ("analysis_result", "model", "create_date") if k in data}
        extra = {k: v for k, v in data.items() if k not in known}
        return cls(**known, extra=extra)


class DailyReportsData(BaseModel):
    pre_market: list[dict[str, Any]] = Field(default_factory=list)
    intraday: list[dict[str, Any]] = Field(default_factory=list)
    post_market: list[dict[str, Any]] = Field(default_factory=list)


class TradingDayData(BaseModel):
    is_trading_day: bool
    date: str
    market: str
    code: str


class TradingDayResponse(BaseModel):
    code: int
    message: str
    data: TradingDayData


class UserBotCode(BaseModel):
    stock_name: str
    code: str
    bot_id: str
    bot_name: str
    bot_type: str


class CapitalDistributionData(BaseModel):
    capital_in_super: float = 0.0
    capital_in_big: float = 0.0
    capital_in_mid: float = 0.0
    capital_in_small: float = 0.0
    capital_out_super: float = 0.0
    capital_out_big: float = 0.0
    capital_out_mid: float = 0.0
    capital_out_small: float = 0.0
    update_time: str | None = None


class BotYesterdayAttitude(BaseModel):
    attitude: str = "neutral"
    analysis_report: str = ""
    bot_id: str = ""
    code: str = ""
    stock_name: str = ""
    date: str | None = None
    language: str = "cn"
    found: bool = True

    @classmethod
    def neutral_default(cls, bot_id: str) -> BotYesterdayAttitude:
        return cls(attitude="neutral", analysis_report="", bot_id=bot_id, found=False)


class PreMarketReportResult(BaseModel):
    report_id: str


class CapitalFlowItem(BaseModel):
    in_flow: float = 0.0
    main_in_flow: float = 0.0
    super_in_flow: float = 0.0
    big_in_flow: float = 0.0
    mid_in_flow: float = 0.0
    sml_in_flow: float = 0.0
    capital_flow_item_time: str | None = None
    last_valid_time: str | None = None


class GeeGooBotClient(BaseClient):
    def get_mcp_analysis(
        self,
        mcp_token: str,
        *,
        name: str,
        code: str,
        prompt_id: str,
        period: McpAnalysisPeriod,
        language: str = "cn",
    ) -> McpAnalysisData:
        payload = self.post(
            "/getMCPAnalysis",
            {
                "mcp_token": mcp_token,
                "name": name,
                "code": code,
                "prompt_id": prompt_id,
                "period": period,
                "language": language,
            },
        )
        data = payload.get("data", {})
        return McpAnalysisData.from_api(data if isinstance(data, dict) else {})

    def get_stock_daily_reports(
        self,
        mcp_token: str,
        code: str,
        report_date: str,
    ) -> DailyReportsData:
        payload = self.post(
            "/getStockDailyReports",
            {
                "mcp_token": mcp_token,
                "code": code,
                "report_date": report_date,
            },
        )
        data = payload.get("data", {})
        if not isinstance(data, dict):
            data = {}
        return DailyReportsData.model_validate(data)

    def check_trading_day(self, mcp_token: str, code: str) -> TradingDayData:
        payload = self.post("/checkTradingDay", {"mcp_token": mcp_token, "code": code})
        return TradingDayResponse.model_validate(payload).data

    def get_report_bot_codes(self, mcp_token: str) -> list[UserBotCode]:
        payload = self.post("/getReportBotCodes", {"mcp_token": mcp_token})
        items = payload.get("data", [])
        return [UserBotCode.model_validate(item) for item in items]

    def get_capital_flow(
        self,
        mcp_token: str,
        code: str,
        *,
        period: CapitalFlowPeriod = "DAY",
        start: str | None = None,
    ) -> list[CapitalFlowItem]:
        body: dict[str, str] = {"mcp_token": mcp_token, "code": code, "period": period}
        if start:
            body["start"] = start
        payload = self.post("/getCapitalFlow", body)
        items = payload.get("data", [])
        return [CapitalFlowItem.model_validate(item) for item in items]

    def get_capital_distribution(self, mcp_token: str, code: str) -> CapitalDistributionData:
        payload = self.post(
            "/getCapitalDistribution",
            {"mcp_token": mcp_token, "code": code},
        )
        data = payload.get("data", {})
        return CapitalDistributionData.model_validate(data if isinstance(data, dict) else {})

    def get_bot_yesterday_attitude(
        self,
        mcp_token: str,
        bot_id: str,
        *,
        language: str = "cn",
    ) -> BotYesterdayAttitude:
        try:
            payload = self.post(
                "/getBotYesterdayAttitude",
                {"mcp_token": mcp_token, "bot_id": bot_id, "language": language},
            )
        except ClientError as exc:
            if exc.api_code == 105 or exc.http_status == 404:
                return BotYesterdayAttitude.neutral_default(bot_id)
            raise
        data = payload.get("data", {})
        if not isinstance(data, dict):
            return BotYesterdayAttitude.neutral_default(bot_id)
        return BotYesterdayAttitude.model_validate({**data, "found": True})

    def create_pre_market_report(self, mcp_token: str, body: dict[str, Any]) -> PreMarketReportResult:
        payload = self.post("/createPreMarketReport", {"mcp_token": mcp_token, **body})
        data = payload.get("data", {})
        report_id = data.get("report_id", "") if isinstance(data, dict) else ""
        return PreMarketReportResult(report_id=str(report_id))
