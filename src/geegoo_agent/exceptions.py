"""Shared exception types."""

from __future__ import annotations


class GeeGooAgentError(Exception):
    """Base exception for GeeGoo Agent."""


class ConfigError(GeeGooAgentError):
    """Invalid or missing configuration."""


class StateStoreError(GeeGooAgentError):
    """State persistence failure."""


class CheckpointError(GeeGooAgentError):
    """Checkpoint read/write failure."""


class SandboxError(GeeGooAgentError):
    """Sandbox policy violation."""


class ModelGatewayError(GeeGooAgentError):
    """LLM gateway failure after retries and fallback."""


class LLMTaskError(GeeGooAgentError):
    """LLM structured task parse or validation failure."""


class ClientError(GeeGooAgentError):
    """HTTP or GeeGoo API business error."""

    def __init__(
        self,
        message: str,
        *,
        code: int | None = None,
        http_status: int | None = None,
    ) -> None:
        super().__init__(message)
        self.api_code = code
        self.http_status = http_status
