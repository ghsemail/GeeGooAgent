"""Application configuration loading."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Literal

from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ConfigError


class LLMConfig(BaseModel):
    provider: Literal["openai", "deepseek", "minimax"] = "openai"
    token_key: str = ""
    model: str = ""
    temperature: float = 0.2
    max_tokens: int = 4096
    thinking: bool | None = None  # None=auto (DeepSeek V4 on); False=off
    reasoning_effort: Literal["low", "medium", "high"] = "high"


class SandboxConfig(BaseModel):
    allowed_hosts: list[str] = Field(default_factory=list)


class AppConfig(BaseModel):
    base_url: str
    api_key: str
    geegoo_url: str
    geegoo_api_key: str
    mcp_token: str
    signal_base_url: str = "http://146.56.225.252:5800"
    output_dir: Path = Path("./data")
    dry_run: bool = False
    feishu_webhook_url: str | None = None
    max_steps: int = 80
    llm: LLMConfig = Field(default_factory=LLMConfig)
    sandbox: SandboxConfig = Field(default_factory=SandboxConfig)

    @property
    def workspace_root(self) -> Path:
        return self.output_dir.resolve()


def load_config(path: str | Path) -> AppConfig:
    config_path = Path(path)
    if not config_path.exists():
        raise ConfigError(f"config file not found: {config_path}")

    try:
        raw = json.loads(config_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise ConfigError(f"invalid JSON in config: {config_path}") from exc

    try:
        config = AppConfig.model_validate(raw)
    except Exception as exc:
        raise ConfigError(f"invalid config fields: {exc}") from exc

    config.output_dir.mkdir(parents=True, exist_ok=True)
    return config
