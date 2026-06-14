"""Tests for BaseClient.post_direct (MCP Common response shapes)."""

from __future__ import annotations

from unittest.mock import MagicMock

import httpx
import pytest

from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.exceptions import ClientError
from geegoo_agent.infra.sandbox import NetworkPolicy


@pytest.fixture
def geegoo_client() -> GeeGooBotClient:
    mock_http = MagicMock(spec=httpx.Client)
    return GeeGooBotClient(
        "http://localhost:5700",
        "test-key",
        NetworkPolicy(["localhost"]),
        client=mock_http,
        retry_wait_seconds=0,
    )


def _mock_response(client: GeeGooBotClient, *, status: int, json_body) -> None:
    response = MagicMock()
    response.status_code = status
    response.content = b"x"
    response.json.return_value = json_body
    client._client.post.return_value = response


@pytest.mark.unit
def test_post_direct_accepts_price_object(geegoo_client: GeeGooBotClient) -> None:
    _mock_response(geegoo_client, status=200, json_body={"price": 99.5})
    assert geegoo_client.post_direct("/getCurrentPrice", {"code": "AAPL.US"}) == {"price": 99.5}


@pytest.mark.unit
def test_post_direct_accepts_search_array(geegoo_client: GeeGooBotClient) -> None:
    items = [{"code": "00700.HK", "name": "腾讯控股"}]
    _mock_response(geegoo_client, status=200, json_body=items)
    assert geegoo_client.post_direct("/searchCode", {"regex": "腾讯"}) == items


@pytest.mark.unit
def test_post_direct_raises_on_business_error(geegoo_client: GeeGooBotClient) -> None:
    _mock_response(geegoo_client, status=200, json_body={"code": 102, "message": "no position"})
    with pytest.raises(ClientError) as exc:
        geegoo_client.post_direct("/getPosition", {"mcp_token": "x", "code": "00700.HK"})
    assert exc.value.api_code == 102
