"""Secrets resolution: environment overrides config file."""

from __future__ import annotations

import os
from collections.abc import Mapping
from typing import Protocol

from geegoo_agent.config import AppConfig
from geegoo_agent.exceptions import ConfigError


class SecretsProvider(Protocol):
    def get(self, key: str) -> str: ...
    def get_optional(self, key: str) -> str | None: ...


_ENV_OVERRIDES: dict[str, str] = {
    "api_key": "SK_API_KEY",
    "geegoo_api_key": "SK_API_KEY",
    "mcp_token": "MCP_TOKEN",
    "llm_token_key": "LLM_TOKEN_KEY",
}


class ConfigSecrets:
    """Resolve secrets with env-first precedence."""

    def __init__(
        self,
        config: AppConfig,
        environ: Mapping[str, str] | None = None,
    ) -> None:
        self._config = config
        self._environ = dict(environ if environ is not None else os.environ)

    def get_optional(self, key: str) -> str | None:
        env_name = _ENV_OVERRIDES.get(key)
        if env_name:
            value = self._environ.get(env_name)
            if value:
                return value
        if key == "llm_token_key":
            return self._config.llm.token_key or None
        return getattr(self._config, key, None)

    @staticmethod
    def _is_placeholder(key: str, value: str) -> bool:
        if not value or not value.strip():
            return True
        upper = value.upper()
        if "REPLACE" in upper:
            return True
        if key in {"api_key", "geegoo_api_key"} and value.strip().endswith("LACE"):
            return True
        return False

    def get(self, key: str) -> str:
        value = self.get_optional(key)
        if value is None or self._is_placeholder(key, value):
            raise ConfigError(f"missing or placeholder secret: {key}")
        return value

    def get_llm_token_key(self) -> str:
        return self.get("llm_token_key")

    def masked(self, key: str) -> str:
        value = self.get_optional(key)
        if not value:
            return "<missing>"
        if len(value) <= 8:
            return "***"
        return f"{value[:4]}...{value[-4:]}"
