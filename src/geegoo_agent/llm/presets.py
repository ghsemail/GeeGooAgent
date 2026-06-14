"""Built-in LLM provider presets (OpenAI-compatible APIs)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

from geegoo_agent.llm.openai_provider import OpenAIProvider

LLMProviderName = Literal["openai", "deepseek", "minimax"]


@dataclass(frozen=True)
class ProviderPreset:
    name: LLMProviderName
    label: str
    base_url: str | None
    default_model: str


PROVIDER_PRESETS: dict[LLMProviderName, ProviderPreset] = {
    "openai": ProviderPreset(
        name="openai",
        label="OpenAI",
        base_url=None,
        default_model="gpt-4o",
    ),
    "deepseek": ProviderPreset(
        name="deepseek",
        label="DeepSeek",
        base_url="https://api.deepseek.com",
        default_model="deepseek-v4-flash",
    ),
    "minimax": ProviderPreset(
        name="minimax",
        label="Minimax",
        base_url="https://api.minimaxi.com/v1",
        default_model="MiniMax-M2.1",
    ),
}

# (model_id, short description) — see https://api-docs.deepseek.com/zh-cn/
PROVIDER_MODELS: dict[LLMProviderName, list[tuple[str, str]]] = {
    "deepseek": [
        ("deepseek-v4-flash", "V4 Flash，快速对话（推荐默认）"),
        ("deepseek-v4-pro", "V4 Pro，复杂推理 / 思考模式"),
        ("deepseek-chat", "旧版 chat（2026/07 弃用，兼容）"),
        ("deepseek-reasoner", "旧版 reasoner（2026/07 弃用，兼容）"),
    ],
    "openai": [
        ("gpt-4o", "GPT-4o"),
        ("gpt-4o-mini", "GPT-4o mini"),
        ("gpt-4-turbo", "GPT-4 Turbo"),
    ],
    "minimax": [
        ("MiniMax-M2.1", "MiniMax M2.1"),
    ],
}


def list_provider_models(provider: LLMProviderName) -> list[tuple[str, str]]:
    return list(PROVIDER_MODELS.get(provider, []))


def current_model(provider: LLMProviderName, model: str | None = None) -> str:
    return resolve_model(provider, model)


def pick_model(provider: LLMProviderName, choice: str, *, current: str | None = None) -> str:
    """Resolve ``/model`` selection: index (1-based), model id, or default."""
    text = choice.strip()
    if not text:
        return current_model(provider, current)

    models = list_provider_models(provider)
    if text.isdigit():
        index = int(text)
        if 1 <= index <= len(models):
            return models[index - 1][0]
        raise ValueError(f"invalid model index: {index} (1-{len(models)})")

    known = {model_id for model_id, _ in models}
    if known and text not in known:
        raise ValueError(f"unknown model for {provider}: {text}")
    return text


def resolve_model(provider: LLMProviderName, model: str | None = None) -> str:
    preset = PROVIDER_PRESETS[provider]
    return (model or "").strip() or preset.default_model


def model_supports_thinking(provider: LLMProviderName, model: str | None = None) -> bool:
    if provider != "deepseek":
        return False
    name = resolve_model(provider, model).lower()
    return "v4" in name or name == "deepseek-reasoner"


def resolve_thinking_enabled(
    provider: LLMProviderName,
    model: str | None,
    *,
    thinking: bool | None,
) -> bool:
    if not model_supports_thinking(provider, model):
        return False
    if thinking is None:
        name = resolve_model(provider, model).lower()
        return "v4" in name
    return thinking


def build_llm_provider(
    provider: LLMProviderName,
    token_key: str,
    *,
    model: str | None = None,
    thinking: bool | None = None,
    reasoning_effort: str = "high",
) -> OpenAIProvider:
    preset = PROVIDER_PRESETS[provider]
    resolved_model = resolve_model(provider, model)
    thinking_enabled = resolve_thinking_enabled(
        provider, resolved_model, thinking=thinking
    )
    return OpenAIProvider(
        resolved_model,
        token_key,
        base_url=preset.base_url,
        thinking_enabled=thinking_enabled,
        reasoning_effort=reasoning_effort if thinking_enabled else None,
    )
