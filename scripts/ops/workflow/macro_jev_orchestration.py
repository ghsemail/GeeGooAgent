#!/usr/bin/env python3
"""Orchestration ledger and synthesis validation for macro JEV daily batch."""

from __future__ import annotations

import json
import os
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from typing import Any

from macro_jev_results_client import JEV_PIPELINE_WORKFLOWS, HttpJsonFn

SHANGHAI_TZ = timezone(timedelta(hours=8))

# Weekday crontab matrix (Asia/Shanghai host TZ unless noted). Weekdays 1-5 only.
MACRO_JEV_WEEKDAY_CRON_SHANGHAI: dict[str, str] = {
    "us": "30 5",
    "commodity_fx": "32 5",
    "bond_us": "35 5",
    "crypto": "38 5",
    "volatility": "40 5",
    "cn_a": "30 15",
    "bond_sh": "0 16",
    "hk": "30 16",
    "overall": "0 17",
}

# Legacy production times (docs / diff only).
MACRO_JEV_LEGACY_WEEKDAY_CRON_SHANGHAI: dict[str, str] = {
    "us": "30 5",
    "commodity_fx": "32 5",
    "bond_us": "35 5",
    "cn_a": "0 7",
    "hk": "2 7",
    "volatility": "10 7",
    "bond_sh": "55 8",
    "overall": "10 9",
    "crypto": "16:15 America/New_York",
}

# Explicit product confirmation.
MORNING_MACRO_POOLS_EXPLICIT = frozenset({"us", "crypto"})
AFTERNOON_MACRO_POOLS_EXPLICIT = frozenset({"cn_a", "hk"})
# Inferred from morning vs afternoon blocks (document as inferred, not user-named).
MORNING_MACRO_POOLS_INFERRED = frozenset({"commodity_fx", "volatility"})
AFTERNOON_MACRO_POOLS_INFERRED = frozenset()  # bond_sh segment; pool tier uses orchestration day

MORNING_CRON_WORKFLOWS = frozenset(
    {"us", "commodity_fx", "bond_us", "crypto", "volatility"}
)
AFTERNOON_CRON_WORKFLOWS = frozenset({"cn_a", "hk", "bond_sh"})

BOND_SEGMENT_WORKFLOWS = frozenset({"bond_sh", "bond_us"})

POOL_RERUN_CRON_WORKFLOWS: dict[str, tuple[str, ...]] = {
    "us": ("us",),
    "commodity_fx": ("commodity_fx",),
    "crypto": ("crypto",),
    "volatility": ("volatility",),
    "cn_a": ("cn_a",),
    "hk": ("hk",),
    "bond": ("bond_us", "bond_sh"),
}


def _parse_ymd(s: str) -> date:
    return datetime.strptime(s.strip(), "%Y-%m-%d").date()


def shanghai_today() -> str:
    return datetime.now(SHANGHAI_TZ).strftime("%Y-%m-%d")


def previous_us_trading_day(orchestration_date: str) -> str:
    """Previous US session calendar day (weekends only; US holidays not modeled)."""
    d = _parse_ymd(orchestration_date) - timedelta(days=1)
    while d.weekday() >= 5:
        d -= timedelta(days=1)
    return d.isoformat()


def orchestration_date_for_run(cron_workflow: str, explicit: str | None = None) -> str:
    _ = cron_workflow
    if explicit and explicit.strip().lower() != "auto":
        return explicit.strip()
    return shanghai_today()


def macro_batch_date_for_cron(cron_workflow: str, orchestration_date: str) -> str:
    """Macro-api `date` for symbol/pool PUT from a staggered cron trigger."""
    if cron_workflow in MORNING_CRON_WORKFLOWS:
        return previous_us_trading_day(orchestration_date)
    if cron_workflow in AFTERNOON_CRON_WORKFLOWS:
        return orchestration_date
    if cron_workflow in ("overall", "all"):
        return orchestration_date
    if cron_workflow in JEV_PIPELINE_WORKFLOWS:
        if cron_workflow in MORNING_MACRO_POOLS_EXPLICIT | MORNING_MACRO_POOLS_INFERRED:
            return previous_us_trading_day(orchestration_date)
        return orchestration_date
    return orchestration_date


def expected_pool_macro_date(pool_workflow: str, orchestration_date: str) -> str:
    """Canonical pool-tier `date` expected when reading ledger for overall."""
    if pool_workflow in MORNING_MACRO_POOLS_EXPLICIT | MORNING_MACRO_POOLS_INFERRED:
        return previous_us_trading_day(orchestration_date)
    if pool_workflow in AFTERNOON_MACRO_POOLS_EXPLICIT:
        return orchestration_date
    if pool_workflow == "bond":
        return orchestration_date
    if pool_workflow == "crypto":
        return previous_us_trading_day(orchestration_date)
    return orchestration_date


def resolve_run_dates(cron_workflow: str, explicit: str | None = None) -> tuple[str, str]:
    """Return (orchestration_date, macro_batch_date) for a cron invocation."""
    orch = orchestration_date_for_run(cron_workflow, explicit)
    macro = macro_batch_date_for_cron(cron_workflow, orch)
    return orch, macro


def ledger_pool_success(ledger: dict[str, dict[str, str]], pool_workflow: str) -> bool:
    row = ledger.get(pool_workflow)
    if not row:
        return False
    return str(row.get("status") or "").strip() == "success"


def pools_needing_rerun(ledger: dict[str, dict[str, str]]) -> list[str]:
    """Pool workflows (seven-pipeline keys) missing a successful ledger slot."""
    out: list[str] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        if not ledger_pool_success(ledger, wf):
            out.append(wf)
    return out


def ledger_path(batch_date: str) -> Path:
    root = Path(os.environ.get("MACRO_JEV_LEDGER_DIR", os.path.expanduser("~/.geegoo")))
    root.mkdir(parents=True, exist_ok=True)
    return root / f"macro_jev_pool_ledger_{batch_date}.json"


def load_pool_ledger(batch_date: str) -> dict[str, dict[str, str]]:
    path = ledger_path(batch_date)
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}
    pools = data.get("pools")
    return pools if isinstance(pools, dict) else {}


def record_pool_ledger(
    batch_date: str,
    pool_workflow: str,
    *,
    result_id: str,
    source_batch_id: str,
    batch_started_at: str,
    status: str = "success",
    cron_workflow: str = "",
) -> None:
    """Append one pool outcome to the day ledger (multi-cron staggered orchestration)."""
    rid = result_id.strip()
    if not rid:
        return
    path = ledger_path(batch_date)
    prev = load_pool_ledger(batch_date)
    doc: dict[str, Any] = {
        "date": batch_date,
        "orchestration_window": batch_date,
        "pools": prev,
    }
    doc["pools"][pool_workflow] = {
        "result_id": rid,
        "source_batch_id": source_batch_id.strip(),
        "batch_started_at": batch_started_at.strip(),
        "status": status.strip() or "success",
        "cron_workflow": cron_workflow.strip() or pool_workflow,
    }
    path.write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")


def ledger_orchestration_date(batch_date: str) -> str:
    """Explicit pipeline date for staggered crons (never cross-day)."""
    return batch_date.strip()


def synthesis_pool_jev_valid(syn: dict[str, Any], *, min_reason_len: int) -> tuple[bool, str]:
    if not syn.get("ok"):
        return False, str(syn.get("error") or "synthesis not ok")
    direction = str(syn.get("jev_direction") or "").strip()
    reason = str(syn.get("jev_reason") or "").strip()
    session_id = str(syn.get("session_id") or "").strip()
    run_id = str(syn.get("run_id") or "").strip()
    if not direction:
        return False, "missing jev_direction from synthesis"
    if len(reason) < min_reason_len:
        return False, f"jev_reason too short (<{min_reason_len})"
    if not session_id or not run_id:
        return False, "missing agent session/run from synthesis"
    return True, ""


def overall_inputs_from_ledger(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
    ledger: dict[str, dict[str, str]],
) -> tuple[list[dict[str, Any]] | None, str]:
    """Build seven overall inputs only from this orchestration ledger (no date-current GET)."""
    if len(ledger) < len(JEV_PIPELINE_WORKFLOWS):
        return None, f"ledger pools {len(ledger)}/{len(JEV_PIPELINE_WORKFLOWS)}"
    headers = {"Authorization": f"Bearer {macro_key}"}
    inputs: list[dict[str, Any]] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        row = ledger.get(wf)
        if not row:
            return None, f"ledger missing pool {wf}"
        rid = str(row.get("result_id") or "").strip()
        if not rid:
            return None, f"ledger missing result_id for {wf}"
        status, data, raw = http_json(
            "GET",
            f"{macro_base.rstrip('/')}/macro/jev/results/by-id/{rid}",
            headers,
            timeout=90,
        )
        if status != 200:
            return None, f"by-id {wf} HTTP {status}: {raw[:120]}"
        block = data.get("data") if isinstance(data, dict) else {}
        result = block.get("result") if isinstance(block, dict) else None
        if not isinstance(result, dict):
            return None, f"by-id {wf} empty result"
        expect_date = expected_pool_macro_date(wf, batch_date)
        if str(result.get("date") or "").strip() != expect_date:
            return None, f"pool {wf} ledger result wrong date"
        if str(result.get("pool_workflow") or "").strip() != wf:
            return None, f"pool {wf} tier mismatch"
        entry_st = str(row.get("status") or "").strip()
        canon_st = str(result.get("status") or "")
        if entry_st and entry_st != "success":
            return None, f"pool {wf} ledger entry status={entry_st}"
        if canon_st != "success":
            return None, f"pool {wf} ledger status={canon_st}"
        inputs.append({"pool_workflow": wf, "result_id": rid})
    return inputs, ""


def pool_synth_rows_from_ledger(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
    ledger: dict[str, dict[str, str]],
) -> tuple[list[dict[str, Any]] | None, str]:
    """Synthesis inputs from ledger-bound pool rows only (no canonical GET latest)."""
    headers = {"Authorization": f"Bearer {macro_key}"}
    rows: list[dict[str, Any]] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        entry = ledger.get(wf)
        if not entry:
            return None, f"ledger missing {wf}"
        rid = str(entry.get("result_id") or "").strip()
        if not rid:
            return None, f"ledger empty result_id for {wf}"
        status, data, raw = http_json(
            "GET",
            f"{macro_base.rstrip('/')}/macro/jev/results/by-id/{rid}",
            headers,
            timeout=90,
        )
        if status != 200:
            return None, f"by-id synth {wf} HTTP {status}: {raw[:120]}"
        block = data.get("data") if isinstance(data, dict) else {}
        result = block.get("result") if isinstance(block, dict) else None
        if not isinstance(result, dict):
            return None, f"synth row missing for {wf}"
        expect_date = expected_pool_macro_date(wf, batch_date)
        if str(result.get("date") or "").strip() != expect_date:
            return None, f"synth {wf} wrong date"
        if str(result.get("status") or "") != "success":
            return None, f"synth {wf} status={result.get('status')}"
        rows.append(
            {
                "label": wf,
                "jev_direction": result.get("jev_direction"),
                "jev_reason": result.get("jev_reason"),
            }
        )
    return rows, ""