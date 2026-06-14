"""Integration tests for MarketClient (httpx mock)."""

from __future__ import annotations

import json
from pathlib import Path

import httpx
import pytest

from geegoo_agent.clients.market import MarketClient
from geegoo_agent.exceptions import ClientError
from geegoo_agent.infra.sandbox import NetworkPolicy

FIXTURES = Path(__file__).resolve().parents[1] / "fixtures" / "geegoo"
BASE_URL = "http://118.195.135.97:5700"
ALLOWED = NetworkPolicy(["118.195.135.97"])


def _load_fixture(name: str) -> dict:
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


@pytest.fixture
def market_client(httpx_mock) -> MarketClient:
    client = httpx.Client()
    return MarketClient(
        BASE_URL,
        "sk-test-key",
        ALLOWED,
        max_retries=3,
        retry_wait_seconds=0,
        client=client,
        sleeper=lambda _s: None,
    )


@pytest.mark.integration
def test_check_trading_day_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/checkTradingDay",
        json=_load_fixture("check_trading_day_ok.json"),
    )
    result = market_client.check_trading_day("mcp-token", "00700.HK")
    assert result.is_trading_day is True
    assert result.market == "HK"
    request = httpx_mock.get_requests()[0]
    assert request.headers["Authorization"] == "Bearer sk-test-key"
    assert json.loads(request.content)["mcp_token"] == "mcp-token"


@pytest.mark.integration
def test_get_report_bot_codes_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getReportBotCodes",
        json=_load_fixture("get_report_bot_codes_ok.json"),
    )
    bots = market_client.get_report_bot_codes("mcp-token")
    assert len(bots) == 1
    assert bots[0].stock_name == "腾讯控股"
    assert bots[0].bot_type == "DCA"
    assert bots[0].bot_id


@pytest.mark.integration
def test_get_capital_flow_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getCapitalFlow",
        json=_load_fixture("get_capital_flow_ok.json"),
    )
    flows = market_client.get_capital_flow("mcp-token", "00700.HK", period="DAY")
    assert len(flows) == 1
    assert flows[0].main_in_flow == -106682800.0
    body = json.loads(httpx_mock.get_requests()[0].content)
    assert body["period"] == "DAY"


@pytest.mark.integration
def test_api_code_not_100_raises_client_error(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/checkTradingDay",
        json={"code": 102, "message": "invalid token"},
        status_code=401,
    )
    with pytest.raises(ClientError) as exc:
        market_client.check_trading_day("bad-token", "00700.HK")
    assert exc.value.api_code == 102


@pytest.mark.integration
def test_http_401_auth_error_raises_client_error(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getReportBotCodes",
        json={"error": "无效的 API Key"},
        status_code=401,
    )
    with pytest.raises(ClientError, match="无效的 API Key"):
        market_client.get_report_bot_codes("mcp-token")


@pytest.mark.integration
def test_server_error_retries_then_raises(market_client: MarketClient, httpx_mock) -> None:
    for _ in range(3):
        httpx_mock.add_response(
            url=f"{BASE_URL}/getCapitalFlow",
            json={"code": 500, "message": "server boom"},
            status_code=500,
        )
    with pytest.raises(ClientError, match="failed after retries"):
        market_client.get_capital_flow("mcp-token", "00700.HK")


@pytest.mark.integration
def test_get_capital_distribution_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getCapitalDistribution",
        json=_load_fixture("get_capital_distribution_ok.json"),
    )
    dist = market_client.get_capital_distribution("mcp-token", "00700.HK")
    assert dist.capital_in_super == 1000000.0
    assert dist.update_time == "2026-04-27 15:59:59"


@pytest.mark.integration
def test_get_bot_yesterday_attitude_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getBotYesterdayAttitude",
        json=_load_fixture("get_bot_yesterday_attitude_ok.json"),
    )
    attitude = market_client.get_bot_yesterday_attitude("mcp-token", "662f3e12ab45cd7890ef1234")
    assert attitude.attitude == "bullish"
    assert attitude.found is True


@pytest.mark.integration
def test_get_bot_yesterday_attitude_404_returns_neutral(
    market_client: MarketClient,
    httpx_mock,
) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getBotYesterdayAttitude",
        json={"code": 105, "message": "未找到昨天的 attitude 记录"},
        status_code=404,
    )
    attitude = market_client.get_bot_yesterday_attitude("mcp-token", "bot-missing")
    assert attitude.attitude == "neutral"
    assert attitude.found is False


@pytest.mark.integration
def test_get_mcp_analysis_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getMCPAnalysis",
        json=_load_fixture("get_mcp_analysis_ok.json"),
    )
    result = market_client.get_mcp_analysis(
        "mcp-token",
        name="恒生指数",
        code="800000.HK",
        prompt_id="69ec7035b9ccd3d9befc6c23",
        period="hourly",
    )
    assert "周线分析" in result.analysis_result


@pytest.mark.integration
def test_create_pre_market_report_ok(market_client: MarketClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/createPreMarketReport",
        json=_load_fixture("create_pre_market_report_ok.json"),
    )
    result = market_client.create_pre_market_report(
        "mcp-token",
        {
            "code": "00700.HK",
            "stock_name": "腾讯控股",
            "bot_id": "bot-1",
            "bot_name": "DCA",
            "bot_type": "DCA",
            "result": "long",
            "confidence": "high",
            "reason": "test",
            "suggestion": "buy",
            "report": "report body",
        },
    )
    assert result.report_id == "680bc8e7f54cf8a14f82a8a2"


@pytest.mark.integration
def test_disallowed_host_raises_before_request() -> None:
    client = MarketClient(
        "http://evil.example.com:5700",
        "sk-test",
        NetworkPolicy(["118.195.135.97"]),
        client=httpx.Client(),
    )
    with pytest.raises(Exception, match="not in allowlist"):
        client.check_trading_day("mcp", "00700.HK")
