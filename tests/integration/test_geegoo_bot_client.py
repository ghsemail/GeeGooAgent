"""Integration tests for GeeGooBotClient."""

from __future__ import annotations

import json
from pathlib import Path

import httpx
import pytest

from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.infra.sandbox import NetworkPolicy

FIXTURES = Path(__file__).resolve().parents[1] / "fixtures" / "geegoo"
BASE_URL = "http://118.195.135.97:5700"
ALLOWED = NetworkPolicy(["118.195.135.97"])


def _load_fixture(name: str) -> dict:
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


@pytest.fixture
def geegoo_bot_client(httpx_mock) -> GeeGooBotClient:
    return GeeGooBotClient(
        BASE_URL,
        "sk-test-key",
        ALLOWED,
        max_retries=3,
        retry_wait_seconds=0,
        client=httpx.Client(),
        sleeper=lambda _s: None,
    )


@pytest.mark.integration
def test_get_mcp_analysis_ok(geegoo_bot_client: GeeGooBotClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getMCPAnalysis",
        json=_load_fixture("get_mcp_analysis_ok.json"),
    )
    result = geegoo_bot_client.get_mcp_analysis(
        "mcp-token",
        name="腾讯控股",
        code="00700.HK",
        prompt_id="69ec7035b9ccd3d9befc6c23",
        period="weekly",
    )
    assert "周线分析" in result.analysis_result


@pytest.mark.integration
def test_get_stock_daily_reports_ok(geegoo_bot_client: GeeGooBotClient, httpx_mock) -> None:
    httpx_mock.add_response(
        url=f"{BASE_URL}/getStockDailyReports",
        json=_load_fixture("get_stock_daily_reports_ok.json"),
    )
    reports = geegoo_bot_client.get_stock_daily_reports(
        "mcp-token",
        "00700.HK",
        "2026-06-05",
    )
    assert len(reports.pre_market) == 1
    assert reports.pre_market[0]["report_id"]
