"""Four-lane lab health + summary build (aligned with GeeGooSignal macro/lab_jev_health.go)."""

from __future__ import annotations

from typing import Any

REQUIRED_LANES = (
    "technical_analysis_jev",
    "index_trend_jev",
    "price_analysis_jev",
    "kline_jev",
)

SKILL_DETAIL_REFS = frozenset({"diagnose_macro", "diagnose_strategy"})

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
    unified = summary.get("jev_unified")
    if not isinstance(unified, dict):
        return True, "jev_unified missing"
    if lane_indicates_failure(unified):
        return True, "jev_unified: " + _lane_text(unified)
    return False, ""


def lab_detail_four_lane_failed(detail: dict[str, Any] | None) -> tuple[bool, str]:
    """lab-detail `detail` object uses the same lane keys as lab_episode_summary."""
    failed, reason = lab_episode_summary_failed(detail)
    if failed:
        return True, reason
    if not isinstance(detail, dict):
        return False, ""
    degraded = detail.get("degraded")
    if isinstance(degraded, list) and degraded:
        tags = [str(x).strip() for x in degraded if str(x).strip()]
        if any("cutoff" in t or "trading_day" in t for t in tags):
            return True, "degraded: " + ";".join(tags[:5])
    return False, ""


def build_lab_episode_summary_from_detail(
    detail: dict[str, Any],
    detail_ref: str,
) -> dict[str, Any] | None:
    if not detail:
        return None
    ref = (detail_ref or "").strip() or str(detail.get("ref") or "").strip()
    if not ref:
        return None
    kline_line = str(detail.get("kline_pattern_summary") or "").strip()
    if not kline_line:
        kp = detail.get("kline_pattern")
        if isinstance(kp, dict):
            kline_line = str(kp.get("summary") or "").strip()
    if kline_line:
        kline_line = "形态 · " + kline_line
    price_line = ""
    pa = detail.get("price_analysis")
    if isinstance(pa, dict):
        price_line = str(pa.get("summary") or "").strip()
        if not price_line:
            rows = pa.get("rows")
            if isinstance(rows, list):
                for row in rows:
                    if not isinstance(row, dict):
                        continue
                    c = str(row.get("conclusion") or "").strip()
                    if c:
                        price_line = c
                        break
    if price_line:
        price_line = "价格 · " + price_line
    out: dict[str, Any] = {"detail_ref": ref}
    for key in ("monitor_quote_cutoff_day", "monitor_quote_cutoff"):
        v = str(detail.get(key) or "").strip()
        if v:
            out[key] = v
    if kline_line:
        out["kline_pattern"] = kline_line
    if price_line:
        out["price_analysis"] = price_line
    for key in (
        "price_analysis_jev",
        "kline_jev",
        "technical_analysis_jev",
        "index_trend_jev",
        "jev_unified",
    ):
        lane = detail.get(key)
        if isinstance(lane, dict) and lane:
            out[key] = lane
    if len(out) <= 1:
        return None
    return out


def is_skill_detail_ref(ref: str) -> bool:
    r = (ref or "").strip()
    if r in SKILL_DETAIL_REFS:
        return True
    return r.startswith("diagnose_")


def direction_skill_to_ledger(unified: dict[str, Any] | None) -> tuple[str, float | None]:
    """Map diagnose skill jev_unified.direction (bull/bear/unclear) to ledger long/short/neutral."""
    if not isinstance(unified, dict):
        return "", None
    raw = str(unified.get("direction") or "").strip().lower()
    conf = unified.get("confidence")
    try:
        conf_f = float(conf) if conf is not None else None
    except (TypeError, ValueError):
        conf_f = None
    if raw == "bull":
        return "long", conf_f
    if raw == "bear":
        return "short", conf_f
    if raw == "unclear":
        return "neutral", conf_f
    direction, conf2 = direction_from_jev_unified(unified)
    if direction:
        return direction, conf2 if conf2 is not None else conf_f
    if unified.get("ok") and not raw:
        return "neutral", conf_f
    return "", conf_f


def build_lab_episode_summary_from_skill_result(
    skill_result: dict[str, Any],
    detail_ref: str = "diagnose_macro",
) -> dict[str, Any] | None:
    """Assemble Monday-facing lab_episode_summary from diagnose_macro skill `result`."""
    if not isinstance(skill_result, dict) or not skill_result:
        return None
    ref = (detail_ref or "").strip() or "diagnose_macro"
    out: dict[str, Any] = {"detail_ref": ref}
    monitor = str(skill_result.get("monitor_session_close") or "").strip()
    if monitor:
        out["monitor_quote_cutoff"] = monitor
    for key in (*REQUIRED_LANES, "jev_unified"):
        lane = skill_result.get(key)
        if isinstance(lane, dict) and lane:
            out[key] = lane
    if len(out) <= 1:
        return None
    return out


def batch_item_from_diagnose_skill_response(
    response: dict[str, Any],
    *,
    detail_ref: str = "diagnose_macro",
    min_reason_len: int = 20,
) -> dict[str, Any]:
    """Map POST /v1/skills/run diagnose_macro JSON to diagnose_symbol_once outcome."""
    session_id = str(response.get("session_id") or "").strip()
    run_id = session_id
    status = str(response.get("status") or "").strip().lower()
    err = str(response.get("error") or "").strip()
    if status != "ok":
        return {
            "ok": False,
            "error": err or f"skill status={status or 'unknown'}",
            "session_id": session_id,
            "run_id": run_id,
            "detail_ref": detail_ref,
        }
    result = response.get("result")
    if not isinstance(result, dict):
        return {
            "ok": False,
            "error": "missing skill result",
            "session_id": session_id,
            "run_id": run_id,
            "detail_ref": detail_ref,
        }
    unified = result.get("jev_unified")
    if not isinstance(unified, dict):
        return {
            "ok": False,
            "error": "jev_unified missing",
            "session_id": session_id,
            "run_id": run_id,
            "detail_ref": detail_ref,
        }
    jev_reason = str(unified.get("recommended_reason") or "").strip()
    direction, conf = direction_skill_to_ledger(unified)
    if not direction:
        direction = "neutral"
    if conf is None:
        conf = 1.0 if direction != "neutral" else 0.5
    lab_summary = build_lab_episode_summary_from_skill_result(result, detail_ref)
    ok_refs, refs_msg = item_refs_complete(
        jev_reason, detail_ref, session_id, run_id, direction, min_reason_len=min_reason_len
    )
    failed, sum_msg = lab_episode_summary_failed(lab_summary)
    if not ok_refs or failed:
        parts: list[str] = []
        if not ok_refs:
            parts.append(refs_msg)
        if failed:
            parts.append(sum_msg)
        return {
            "ok": False,
            "error": " · ".join(parts) or "incomplete jev",
            "session_id": session_id,
            "run_id": run_id,
            "jev_reason": jev_reason,
            "detail_ref": detail_ref,
            "direction": direction,
        }
    return {
        "ok": True,
        "session_id": session_id,
        "run_id": run_id,
        "jev_reason": jev_reason,
        "detail_ref": detail_ref,
        "direction": direction,
        "confidence": conf,
        "lab_episode_summary": lab_summary,
    }


def direction_from_jev_unified(unified: dict[str, Any] | None) -> tuple[str, float | None]:
    if not isinstance(unified, dict):
        return "", None
    choice = str(
        unified.get("choice")
        or unified.get("direction")
        or unified.get("recommended_choice")
        or ""
    ).strip().lower()
    conf = unified.get("confidence")
    try:
        conf_f = float(conf) if conf is not None else None
    except (TypeError, ValueError):
        conf_f = None
    if choice in ("long", "short", "neutral"):
        return choice, conf_f
    if choice in ("bull", "bear"):
        return ("long" if choice == "bull" else "short"), conf_f
    if any(x in choice for x in ("多", "涨", "bull")):
        return "long", conf_f
    if any(x in choice for x in ("空", "跌", "bear")):
        return "short", conf_f
    return "", conf_f


def item_refs_complete(
    jev_reason: str,
    detail_ref: str,
    session_id: str,
    run_id: str,
    direction: str,
    *,
    min_reason_len: int = 20,
) -> tuple[bool, str]:
    if len((jev_reason or "").strip()) < min_reason_len:
        return False, f"jev_reason<{min_reason_len}"
    ref = (detail_ref or "").strip()
    if is_skill_detail_ref(ref):
        if not ref:
            return False, "missing skill detail_ref"
    elif not (ref.startswith("buy:") or ref.startswith("sell:")):
        return False, "missing buy:/sell: detail_ref"
    if not (session_id or "").strip() or not (run_id or "").strip():
        return False, "missing session/run"
    if not (direction or "").strip():
        return False, "missing direction"
    return True, ""