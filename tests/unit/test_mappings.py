"""Unit tests for business mappings."""

from __future__ import annotations

import pytest

from geegoo_agent.tools.mappings import attitude_to_result, format_capital_distribution, is_a_share


@pytest.mark.unit
def test_attitude_bullish_maps_to_long() -> None:
    assert attitude_to_result("bullish") == "long"


@pytest.mark.unit
def test_attitude_bearish_maps_to_short() -> None:
    assert attitude_to_result("bearish") == "short"


@pytest.mark.unit
def test_attitude_unknown_defaults_neutral() -> None:
    assert attitude_to_result("unknown") == "neutral"


@pytest.mark.unit
@pytest.mark.parametrize(
    ("code", "expected"),
    [
        ("600519.SH", True),
        ("000001.SZ", True),
        ("00700.HK", False),
        ("AAPL.US", False),
    ],
)
def test_is_a_share(code: str, expected: bool) -> None:
    assert is_a_share(code) is expected


@pytest.mark.unit
def test_format_capital_distribution_includes_update_time() -> None:
    text = format_capital_distribution(
        {
            "capital_in_super": 1e9,
            "capital_out_super": 8e8,
            "capital_in_big": 5e8,
            "capital_out_big": 4e8,
            "capital_in_mid": 3e8,
            "capital_out_mid": 2e8,
            "capital_in_small": 1e8,
            "capital_out_small": 1e8,
            "update_time": "2026-06-05 15:59:59",
        }
    )
    assert "超大单" in text
    assert "更新时间：2026-06-05 15:59:59" in text
