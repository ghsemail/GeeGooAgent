"""POST /v1/skills/run diagnose_macro_pool — pool JEV synthesis (no Agent Loop)."""

from __future__ import annotations

import re
from typing import Any, Callable

from macro_jev_synthesis_format import format_pool_jev_reason

ISO_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
POOL_WORKFLOWS = frozenset(
    {"cn_a", "hk", "us", "commodity_fx", "bond", "crypto", "volatility"}
)
VALID_DIRECTIONS = frozenset({"long", "short", "neutral"})

HttpJsonFn = Callable[..., tuple[int, Any, str]]


class PoolSkillPackError(ValueError):
    """Workflow payload failed fail-closed validation."""


def truncate_reason_excerpt(text: str, *, max_len: int = 800) -> str:
    raw = (text or "").strip()
    if len(raw) <= max_len:
        return raw
    return raw[: max_len - 1].rstrip() + "…"


def validate_pool_skill_pack(
    *,
    batch_date: str,
    pool_workflow: str,
    source_batch_id: str,
    symbol_rows: list[dict[str, Any]],
    symbol_result_ids: dict[str, str] | None,
    min_reason_len: int,
) -> list[dict[str, Any]]:
    """Build macro_pool.symbols[]; raises PoolSkillPackError when incomplete."""
    date = (batch_date or "").strip()
    if not ISO_DATE_RE.match(date):
        raise PoolSkillPackError("invalid batch_date")
    wf = (pool_workflow or "").strip()
    if wf not in POOL_WORKFLOWS:
        raise PoolSkillPackError(f"invalid pool_workflow={wf!r}")
    sbid = (source_batch_id or "").strip()
    if not sbid:
        raise PoolSkillPackError("source_batch_id required")
    if not symbol_rows:
        raise PoolSkillPackError("no symbol rows")
    rid_map = {k.upper(): v.strip() for k, v in (symbol_result_ids or {}).items() if v}
    symbols_out: list[dict[str, Any]] = []
    seen: set[str] = set()
    for row in symbol_rows:
        sym = str(row.get("symbol") or "").strip()
        if not sym:
            raise PoolSkillPackError("missing symbol")
        upper = sym.upper()
        if upper in seen:
            raise PoolSkillPackError(f"duplicate symbol {sym}")
        seen.add(upper)
        if not row.get("success", True):
            raise PoolSkillPackError(f"symbol not success: {sym}")
        direction = str(row.get("jev_direction") or "").strip().lower()
        if direction not in VALID_DIRECTIONS:
            raise PoolSkillPackError(f"invalid jev_direction for {sym}")
        reason = str(row.get("jev_reason") or "").strip()
        if len(reason) < min_reason_len:
            raise PoolSkillPackError(f"jev_reason too short for {sym}")
        result_id = str(row.get("result_id") or rid_map.get(upper) or "").strip()
        if not result_id:
            raise PoolSkillPackError(f"missing result_id for {sym}")
        entry: dict[str, Any] = {
            "symbol": sym,
            "result_id": result_id,
            "jev_direction": direction,
            "jev_reason": truncate_reason_excerpt(reason),
        }
        conf = row.get("confidence")
        try:
            if conf is not None:
                entry["confidence"] = float(conf)
        except (TypeError, ValueError):
            pass
        symbols_out.append(entry)
    return symbols_out


def build_macro_pool_skill_body(
    *,
    batch_date: str,
    pool_workflow: str,
    source_batch_id: str,
    title: str,
    symbol_rows: list[dict[str, Any]],
    symbol_result_ids: dict[str, str] | None = None,
    min_reason_len: int = 20,
) -> dict[str, Any]:
    symbols = validate_pool_skill_pack(
        batch_date=batch_date,
        pool_workflow=pool_workflow,
        source_batch_id=source_batch_id,
        symbol_rows=symbol_rows,
        symbol_result_ids=symbol_result_ids,
        min_reason_len=min_reason_len,
    )
    return {
        "skill": "diagnose_macro_pool",
        "macro_pool": {
            "date": batch_date.strip(),
            "pool_workflow": pool_workflow.strip(),
            "source_batch_id": source_batch_id.strip(),
            "title": (title or "").strip(),
            "symbols": symbols,
        },
    }


def run_diagnose_macro_pool_skill(
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


def pool_jev_from_diagnose_macro_pool_skill_response(
    response: dict[str, Any],
    *,
    min_reason_len: int = 20,
) -> dict[str, Any]:
    """Map diagnose_macro_pool skill JSON to synthesize_pool_jev_valid shape."""
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
        return {
            "ok": False,
            "error": "missing jev_direction",
            "session_id": session_id,
            "run_id": run_id,
        }
    if len(reason) < min_reason_len:
        return {
            "ok": False,
            "error": "jev_reason too short",
            "session_id": session_id,
            "run_id": run_id,
        }
    out: dict[str, Any] = {
        "ok": True,
        "jev_direction": direction,
        "jev_reason": reason,
        "session_id": session_id,
        "run_id": run_id,
        "detail_ref": str(result.get("detail_ref") or "diagnose_macro_pool"),
    }
    if conf_f is not None:
        out["confidence"] = conf_f
    if "conflict" in result:
        out["conflict"] = bool(result.get("conflict"))
    return out


def synthesize_pool_jev_with_skill_retries(
    agent_base: str,
    runtime_key: str,
    *,
    batch_date: str,
    pool_workflow: str,
    source_batch_id: str,
    title: str,
    symbol_rows: list[dict[str, Any]],
    symbol_result_ids: dict[str, str] | None,
    http_json: HttpJsonFn,
    min_reason_len: int = 20,
    max_retries: int = 2,
    retry_delay_sec: float = 10.0,
) -> dict[str, Any]:
    import time

    try:
        body = build_macro_pool_skill_body(
            batch_date=batch_date,
            pool_workflow=pool_workflow,
            source_batch_id=source_batch_id,
            title=title,
            symbol_rows=symbol_rows,
            symbol_result_ids=symbol_result_ids,
            min_reason_len=min_reason_len,
        )
    except PoolSkillPackError as exc:
        return {"ok": False, "error": str(exc)}
    last: dict[str, Any] = {"ok": False, "error": "no attempt"}
    for attempt in range(max_retries + 1):
        if attempt > 0:
            time.sleep(retry_delay_sec)
        raw_resp = run_diagnose_macro_pool_skill(
            agent_base, runtime_key, body, http_json,
        )
        if raw_resp.get("ok") is False and "status" not in raw_resp:
            last = {"ok": False, "error": raw_resp.get("error"), "attempt": attempt + 1}
            continue
        syn = pool_jev_from_diagnose_macro_pool_skill_response(
            raw_resp, min_reason_len=min_reason_len,
        )
        syn["attempt"] = attempt + 1
        if syn.get("ok"):
            return syn
        last = {**syn, "attempt": attempt + 1}
    return last