"""Unit tests for secrets resolution."""

from __future__ import annotations

import pytest

from geegoo_agent.config import AppConfig
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.infra.secrets import ConfigSecrets


def _sample_config() -> AppConfig:
    return AppConfig(
        base_url="http://x:5700",
        api_key="sk-from-file",
        geegoo_url="http://x:5700",
        geegoo_api_key="sk-from-file",
        mcp_token="mcp-from-file",
    )


@pytest.mark.unit
def test_secrets_reads_from_config() -> None:
    secrets = ConfigSecrets(_sample_config(), environ={})
    assert secrets.get("api_key") == "sk-from-file"
    assert secrets.get("mcp_token") == "mcp-from-file"


@pytest.mark.unit
def test_secrets_env_overrides_config() -> None:
    secrets = ConfigSecrets(
        _sample_config(),
        environ={"SK_API_KEY": "sk-from-env", "MCP_TOKEN": "mcp-from-env"},
    )
    assert secrets.get("api_key") == "sk-from-env"
    assert secrets.get("mcp_token") == "mcp-from-env"


@pytest.mark.unit
def test_secrets_llm_token_key_from_config() -> None:
    config = _sample_config()
    config = config.model_copy(update={"llm": config.llm.model_copy(update={"token_key": "sk-llm"})})
    secrets = ConfigSecrets(config, environ={})
    assert secrets.get_llm_token_key() == "sk-llm"


@pytest.mark.unit
def test_secrets_llm_token_key_env_overrides_config() -> None:
    config = _sample_config()
    config = config.model_copy(update={"llm": config.llm.model_copy(update={"token_key": "from-file"})})
    secrets = ConfigSecrets(config, environ={"LLM_TOKEN_KEY": "from-env"})
    assert secrets.get_llm_token_key() == "from-env"


@pytest.mark.unit
def test_secrets_placeholder_raises() -> None:
    config = AppConfig(
        base_url="http://x:5700",
        api_key="sk-REPLACE",
        geegoo_url="http://x:5700",
        geegoo_api_key="sk-REPLACE",
        mcp_token="REPLACE",
    )
    secrets = ConfigSecrets(config, environ={})
    with pytest.raises(ConfigError, match="placeholder"):
        secrets.get("api_key")
    with pytest.raises(ConfigError, match="placeholder"):
        secrets.get("geegoo_api_key")
    with pytest.raises(ConfigError, match="placeholder"):
        secrets.get("mcp_token")


@pytest.mark.unit
def test_secrets_masked_does_not_leak_full_key() -> None:
    secrets = ConfigSecrets(_sample_config(), environ={})
    masked = secrets.masked("api_key")
    assert "sk-from-file" not in masked
    assert "..." in masked
