"""Unit tests for CostManager."""

from __future__ import annotations

import pytest

from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.types import TokenUsage


@pytest.mark.unit
def test_cost_manager_session_total() -> None:
    cost = CostManager()
    cost.record("s1", 1, TokenUsage(10, 5, "gpt-4o", 0.01))
    cost.record("s1", 2, TokenUsage(20, 10, "gpt-4o", 0.02))
    total = cost.session_total("s1")
    assert total.prompt_tokens == 30
    assert total.completion_tokens == 15
    assert total.estimated_usd == pytest.approx(0.03)


@pytest.mark.unit
def test_cost_manager_unknown_session_returns_zero() -> None:
    cost = CostManager()
    total = cost.session_total("missing")
    assert total.prompt_tokens == 0
    assert total.completion_tokens == 0
