"""Four-lane JEV health checks for macro batch (mirrors GeeGooSignal/scripts/macro_lab_jev_health.py)."""

from __future__ import annotations

from typing import Any

REQUIRED_LANES = (
    "technical_analysis_jev",
    "index_trend_jev",
    "price_analysis_jev",
    "kline_jev",
)

FAIL_FRAGMENTS = (
    "deadline exceeded",
    "deadline_exceeded",
    "context deadline",
    "timeout",
    "timed out",
    "lane failed",
    "bar index out of range",
    "macro_trading_day",
    "macro_monitor_cutoff",
)


def _lane_text(lane: dict[str, Any]) -> str:
    for key in ("error", "reason", "jev_reason", "message"):
        val = lane.get(key)
        if isinstance(val, str) and val.strip():
            return val.strip()
    return ""


def lane_indicates_failure(lane: dict[str, Any] | None) -> bool:
    if not lane:
        return True
    ok = lane.get("ok")
    if ok is False or ok == "false" or ok == 0:
        return True
    txt = _lane_text(lane).lower()
    return any(frag in txt for frag in FAIL_FRAGMENTS)


def lab_episode_summary_failed(summary: dict[str, Any] | None) -> tuple[bool, str]:
    if not summary:
        return True, "lab_episode_summary missing"
    unified = summary.get("jev_unified")
    if not isinstance(unified, dict):
        return True, "jev_unified missing"
    if lane_indicates_failure(unified):
        return True, "jev_unified: " + _lane_text(unified)
    missing: list[str] = []
    for key in REQUIRED_LANES:
        lane = summary.get(key)
        if not isinstance(lane, dict):
            missing.append(key)
            continue
        if lane_indicates_failure(lane):
            return True, f"{key}: " + _lane_text(lane)
    if missing:
        return True, "lanes missing: " + ",".join(missing)
    return False, ""


def lab_detail_four_lane_failed(detail: dict[str, Any] | None) -> tuple[bool, str]:
    """lab-detail `detail` object uses the same lane keys as lab_episode_summary."""
    failed, reason = lab_episode_summary_failed(detail)
    if failed:
        return True, reason
    degraded = detail.get("degraded") if isinstance(detail, dict) else None
    if isinstance(degraded, list) and degraded:
        tags = [str(x).strip() for x in degraded if str(x).strip()]
        if any("cutoff" in t or "trading_day" in t for t in tags):
            return True, "degraded: " + ";".join(tags[:5])
    return False, ""