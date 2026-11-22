"""POST /v1/skills/run diagnose_macro_overall — macro overall JEV (no Agent Loop)."""

from __future__ import annotations

import re
from typing import Any, Callable

from macro_jev_synthesis_format import format_pool_jev_reason

ISO_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
POOL_WORKFLOWS = frozenset(
    {"cn_a", "hk", "us", "commodity_fx", "bond", "crypto", "volatility"}
)
VALID_DIRECTIONS = frozenset({"long", "short", "neutral"})
CHOICE_LABELS = frozenset({"看多", "看空", "不明确"})

HttpJsonFn = Callable[..., tuple[int, Any, str]]


class OverallSkillPackError(ValueError):
    pass


def validate_overall_pools(
    pools: list[dict[str, Any]],
    *,
    min_reason_len: int,
) -> list[dict[str, Any]]:
    if not pools:
        raise OverallSkillPackError("pools required")
    if len(pools) > 7:
        raise OverallSkillPackError("too many pools")
    out: list[dict[str, Any]] = []
    seen: set[str] = set()
    for row in pools:
        wf = str(row.get("pool_workflow") or row.get("label") or "").strip()
        if wf not in POOL_WORKFLOWS:
            raise OverallSkillPackError(f"invalid pool_workflow={wf!r}")
        if wf in seen:
            raise OverallSkillPackError(f"duplicate pool_workflow {wf}")
        seen.add(wf)
        direction = str(row.get("jev_direction") or "").strip().lower()
        if direction not in VALID_DIRECTIONS:
            raise OverallSkillPackError(f"invalid jev_direction for {wf}")
        reason = str(row.get("jev_reason") or "").strip()
        if len(reason) < min_reason_len:
            raise OverallSkillPackError(f"jev_reason too short for {wf}")
        out.append({
            "pool_workflow": wf,
            "jev_direction": direction,
            "jev_reason": reason,
        })
    return out


def build_macro_overall_skill_body(
    *,
    orchestration_date: str,
    pools: list[dict[str, Any]],
    min_reason_len: int = 20,
) -> dict[str, Any]:
    date = (orchestration_date or "").strip()
    if not ISO_DATE_RE.match(date):
        raise OverallSkillPackError("invalid date")
    pool_rows = validate_overall_pools(pools, min_reason_len=min_reason_len)
    return {
        "skill": "diagnose_macro_overall",
        "macro_overall": {
            "date": date,
            "pools": pool_rows,
        },
    }


def run_diagnose_macro_overall_skill(
    agent_base: str,
    runtime_key: str,
    body: dict[str, Any],
    http_json: HttpJsonFn,
    *,
    timeout: int = 300,
) -> dict[str, Any]:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
    }
    status, data, raw = http_json(
        "POST",
        f"{agent_base.rstrip('/')}/v1/skills/run",
        headers,
        body,
        timeout=timeout,
    )
    if status != 200:
        return {"ok": False, "error": f"skills/run HTTP {status}: {raw[:400]}"}
    if not isinstance(data, dict):
        return {"ok": False, "error": "skills/run invalid json"}
    return data


def parse_overall_choice_distribution(raw: Any) -> list[dict[str, Any]] | None:
    if not isinstance(raw, list) or not raw:
        return None
    out: list[dict[str, Any]] = []
    for item in raw:
        if not isinstance(item, dict):
            return None
        label = str(item.get("label") or "").strip()
        if label not in CHOICE_LABELS:
            return None
        try:
            prob_f = float(item.get("probability"))
        except (TypeError, ValueError):
            return None
        if prob_f < 0.0 or prob_f > 1.0:
            return None
        out.append({"label": label, "probability": prob_f})
    return out if out else None


def overall_jev_from_skill_response(
    response: dict[str, Any],
    *,
    min_reason_len: int = 20,
) -> dict[str, Any]:
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
        }
    result = response.get("result")
    if not isinstance(result, dict):
        return {"ok": False, "error": "missing skill result", "session_id": session_id}
    direction = str(result.get("jev_direction") or "").strip().lower()
    reason = format_pool_jev_reason(str(result.get("jev_reason") or ""))
    conf = result.get("confidence")
    try:
        conf_f = float(conf) if conf is not None else None
    except (TypeError, ValueError):
        conf_f = None
    if direction not in VALID_DIRECTIONS:
        return {"ok": False, "error": "missing jev_direction", "session_id": session_id}
    if len(reason) < min_reason_len:
        return {"ok": False, "error": "jev_reason too short", "session_id": session_id}
    choices = parse_overall_choice_distribution(result.get("choices"))
    if not choices:
        return {"ok": False, "error": "missing or invalid choices", "session_id": session_id}
    out: dict[str, Any] = {
        "ok": True,
        "jev_direction": direction,
        "jev_reason": reason,
        "choices": choices,
        "session_id": session_id,
        "run_id": run_id,
        "detail_ref": str(result.get("detail_ref") or "diagnose_macro_overall"),
    }
    if conf_f is not None:
        out["confidence"] = conf_f
    return out


def synthesize_overall_jev_with_skill_retries(
    agent_base: str,
    runtime_key: str,
    *,
    orchestration_date: str,
    pools: list[dict[str, Any]],
    http_json: HttpJsonFn,
    min_reason_len: int = 20,
    max_retries: int = 2,
    retry_delay_sec: float = 10.0,
) -> dict[str, Any]:
    import time

    try:
        body = build_macro_overall_skill_body(
            orchestration_date=orchestration_date,
            pools=pools,
            min_reason_len=min_reason_len,
        )
    except OverallSkillPackError as exc:
        return {"ok": False, "error": str(exc)}
    last: dict[str, Any] = {"ok": False, "error": "no attempt"}
    for attempt in range(max_retries + 1):
        if attempt > 0:
            time.sleep(retry_delay_sec)
        raw_resp = run_diagnose_macro_overall_skill(
            agent_base, runtime_key, body, http_json,
        )
        if raw_resp.get("ok") is False and "status" not in raw_resp:
            last = {"ok": False, "error": raw_resp.get("error"), "attempt": attempt + 1}
            continue
        syn = overall_jev_from_skill_response(raw_resp, min_reason_len=min_reason_len)
        syn["attempt"] = attempt + 1
        if syn.get("ok"):
            return syn
        last = {**syn, "attempt": attempt + 1}
    return last