"""Unit tests for LLM provider presets."""

from __future__ import annotations

import pytest

from geegoo_agent.llm.openai_provider import OpenAIProvider
from geegoo_agent.llm.presets import build_llm_provider, pick_model, resolve_model


@pytest.mark.unit
@pytest.mark.parametrize(
    ("provider", "expected_model", "expected_base"),
    [
        ("openai", "gpt-4o", None),
        ("deepseek", "deepseek-v4-flash", "https://api.deepseek.com"),
        ("minimax", "MiniMax-M2.1", "https://api.minimaxi.com/v1"),
    ],
)
def test_build_llm_provider_defaults(provider, expected_model, expected_base) -> None:
    llm = build_llm_provider(provider, "test-key")
    assert isinstance(llm, OpenAIProvider)
    assert llm.model == expected_model
    assert llm._base_url == expected_base


@pytest.mark.unit
def test_resolve_model_override() -> None:
    assert resolve_model("deepseek", "deepseek-reasoner") == "deepseek-reasoner"


@pytest.mark.unit
def test_pick_model_by_index_and_id() -> None:
    assert pick_model("deepseek", "2") == "deepseek-v4-pro"
    assert pick_model("deepseek", "deepseek-v4-flash") == "deepseek-v4-flash"


@pytest.mark.unit
def test_pick_model_rejects_unknown() -> None:
    with pytest.raises(ValueError, match="unknown model"):
        pick_model("deepseek", "not-a-model")
