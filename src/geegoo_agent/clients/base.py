"""HTTP base client for GeeGoo APIs."""

from __future__ import annotations

import time
from collections.abc import Callable
from typing import Any

import httpx

from geegoo_agent.exceptions import ClientError
from geegoo_agent.infra.sandbox import NetworkPolicy

Retryable = Callable[[], dict[str, Any]]


class BaseClient:
    def __init__(
        self,
        base_url: str,
        api_key: str,
        network: NetworkPolicy,
        *,
        timeout: float = 60.0,
        max_retries: int = 3,
        retry_wait_seconds: float = 5.0,
        client: httpx.Client | None = None,
        sleeper: Callable[[float], None] = time.sleep,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.network = network
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_wait_seconds = retry_wait_seconds
        self._client = client or httpx.Client(timeout=timeout)
        self._sleep = sleeper
        self._owns_client = client is None

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def post(self, path: str, body: dict[str, Any]) -> dict[str, Any]:
        url = f"{self.base_url}{path}"
        self.network.assert_host_allowed(url)

        last_error: Exception | None = None
        for attempt in range(self.max_retries):
            try:
                response = self._client.post(
                    url,
                    json=body,
                    headers={
                        "Authorization": f"Bearer {self.api_key}",
                        "Content-Type": "application/json",
                    },
                )
                if response.status_code >= 500:
                    raise ClientError(
                        f"server error {response.status_code} for {path}",
                        http_status=response.status_code,
                    )
                data = response.json() if response.content else {}
                if response.status_code == 401 and "error" in data and "code" not in data:
                    raise ClientError(
                        str(data.get("error", "unauthorized")),
                        http_status=401,
                    )
                code = data.get("code")
                if code != 100:
                    raise ClientError(
                        data.get("message", f"api error code {code}"),
                        code=code,
                        http_status=response.status_code,
                    )
                return data
            except (httpx.TimeoutException, httpx.TransportError, ClientError) as exc:
                if isinstance(exc, ClientError) and exc.http_status not in {None, 500} and (
                    exc.http_status or 0
                ) < 500:
                    raise
                if isinstance(exc, ClientError) and exc.api_code is not None:
                    raise
                last_error = exc
                if attempt < self.max_retries - 1:
                    self._sleep(self.retry_wait_seconds)

        raise ClientError(f"request failed after retries: {last_error}") from last_error

    def post_direct(self, path: str, body: dict[str, Any]) -> Any:
        """POST and return parsed JSON without requiring ``code == 100``.

        Used for MCP Common endpoints that return a bare object (e.g. ``price``)
        or array (e.g. ``searchCode``). If the body is a dict with ``code`` and
        it is not 100, still raises :class:`ClientError`.
        """
        url = f"{self.base_url}{path}"
        self.network.assert_host_allowed(url)

        last_error: Exception | None = None
        for attempt in range(self.max_retries):
            try:
                response = self._client.post(
                    url,
                    json=body,
                    headers={
                        "Authorization": f"Bearer {self.api_key}",
                        "Content-Type": "application/json",
                    },
                )
                if response.status_code >= 500:
                    raise ClientError(
                        f"server error {response.status_code} for {path}",
                        http_status=response.status_code,
                    )
                if response.status_code == 400:
                    data = response.json() if response.content else {}
                    if isinstance(data, dict):
                        raise ClientError(
                            str(data.get("message", data)),
                            code=data.get("code"),
                            http_status=400,
                        )
                data = response.json() if response.content else {}
                code = data.get("code") if isinstance(data, dict) else None
                if code is not None and code != 100:
                    raise ClientError(
                        str(data.get("message", f"api error code {data.get('code')}")),
                        code=data.get("code"),
                        http_status=response.status_code,
                    )
                if response.status_code == 401 and isinstance(data, dict) and "error" in data:
                    raise ClientError(
                        str(data.get("error", "unauthorized")),
                        http_status=401,
                    )
                return data
            except (httpx.TimeoutException, httpx.TransportError, ClientError) as exc:
                if isinstance(exc, ClientError) and exc.http_status not in {None, 500} and (
                    exc.http_status or 0
                ) < 500:
                    raise
                if isinstance(exc, ClientError) and exc.api_code is not None:
                    raise
                last_error = exc
                if attempt < self.max_retries - 1:
                    self._sleep(self.retry_wait_seconds)

        raise ClientError(f"request failed after retries: {last_error}") from last_error
