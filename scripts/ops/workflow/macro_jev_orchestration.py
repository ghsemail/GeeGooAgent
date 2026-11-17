#!/usr/bin/env python3
"""Orchestration ledger and synthesis validation for macro JEV daily batch."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any

from macro_jev_results_client import JEV_PIPELINE_WORKFLOWS, HttpJsonFn


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
        if str(result.get("date") or "").strip() != batch_date:
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
        if str(result.get("date") or "").strip() != batch_date:
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