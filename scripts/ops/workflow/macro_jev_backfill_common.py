#!/usr/bin/env python3
"""Shared guards, state, and validation for macro JEV audit backfill (see docs/MACRO_JEV_BACKFILL_MINIMAL_LOOP.md)."""

from __future__ import annotations

import json
import os
import re
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any, Callable
from urllib.parse import urlparse

from macro_jev_audit_schema import (
    DayAuditReport,
    JEV_PIPELINE_WORKFLOWS,
    classify_day_export,
)
from macro_jev_results_client import HttpJsonFn

DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")

BACKFILL_STATE_ENV = "MACRO_JEV_BACKFILL_STATE_DIR"


def backfill_state_path(batch_date: str) -> Path:
    root = Path(os.environ.get(BACKFILL_STATE_ENV, os.path.expanduser("~/.geegoo")))
    root.mkdir(parents=True, exist_ok=True)
    return root / f"macro_jev_backfill_state_{batch_date}.json"


@dataclass
class BackfillState:
    batch_date: str
    export_fingerprint: str = ""
    phase_a: dict[str, dict[str, Any]] = field(default_factory=dict)
    phase_b: dict[str, dict[str, Any]] = field(default_factory=dict)
    phase_c: dict[str, Any] = field(default_factory=dict)
    failures: list[dict[str, Any]] = field(default_factory=list)


def load_backfill_state(batch_date: str) -> BackfillState:
    path = backfill_state_path(batch_date)
    if not path.is_file():
        return BackfillState(batch_date=batch_date)
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return BackfillState(batch_date=batch_date)
    return BackfillState(
        batch_date=batch_date,
        export_fingerprint=str(raw.get("export_fingerprint") or ""),
        phase_a=dict(raw.get("phase_a") or {}),
        phase_b=dict(raw.get("phase_b") or {}),
        phase_c=dict(raw.get("phase_c") or {}),
        failures=list(raw.get("failures") or []),
    )


def save_backfill_state(state: BackfillState) -> None:
    path = backfill_state_path(state.batch_date)
    doc = {
        "batch_date": state.batch_date,
        "export_fingerprint": state.export_fingerprint,
        "phase_a": state.phase_a,
        "phase_b": state.phase_b,
        "phase_c": state.phase_c,
        "failures": state.failures,
    }
    path.write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")


def symbol_state_key(pool_workflow: str, symbol: str, source_batch_id: str) -> str:
    return f"{pool_workflow}:{symbol.upper()}:{source_batch_id}"


def export_fingerprint(export: dict[str, Any]) -> str:
    """Stable hash substitute: sorted run ids."""
    runs = export.get("pool_runs_by_workflow") or {}
    parts = []
    for wf in sorted(runs.keys()):
        run = runs[wf]
        if isinstance(run, dict):
            parts.append(f"{wf}={run.get('id') or run.get('_id') or ''}")
    return "|".join(parts)


def parse_export_file(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError("export root must be object")
    return data


def validate_export_dates(export: dict[str, Any], batch_date: str) -> list[str]:
    errs: list[str] = []
    if not DATE_RE.match(batch_date):
        errs.append(f"invalid batch_date {batch_date!r}")
    exp_date = str(export.get("batch_date") or "").strip()
    if exp_date and exp_date != batch_date:
        errs.append(f"export batch_date={exp_date} != --date {batch_date}")
    runs = export.get("pool_runs_by_workflow")
    if not isinstance(runs, dict) or not runs:
        errs.append("pool_runs_by_workflow missing or empty")
        return errs
    for wf, run in runs.items():
        if wf not in JEV_PIPELINE_WORKFLOWS:
            errs.append(f"unknown pool workflow key {wf!r}")
        if not isinstance(run, dict):
            errs.append(f"run for {wf} not an object")
            continue
        rid = str(run.get("id") or run.get("_id") or "").strip()
        if not rid:
            errs.append(f"{wf}: missing run id")
        rdate = str(run.get("date") or "").strip()
        if rdate and rdate != batch_date:
            errs.append(f"{wf}: run.date {rdate} != {batch_date}")
        started = str(run.get("started_at") or "").strip()
        if not started:
            errs.append(f"{wf}: missing started_at")
        finished = str(run.get("finished_at") or "").strip()
        if not finished:
            errs.append(f"{wf}: missing finished_at (not a finished audit run)")
    return errs


def audit_export(export: dict[str, Any], batch_date: str) -> DayAuditReport:
    runs = export.get("pool_runs_by_workflow") or {}
    for wf, run in list(runs.items()):
        if isinstance(run, dict) and not run.get("id"):
            run = {**run, "id": str(run.get("_id") or "")}
            runs[wf] = run
    return classify_day_export(
        batch_date=batch_date,
        pool_runs_by_workflow=runs,
        overall_runs=list(export.get("overall_runs") or []),
        bond_merged=bool(export.get("bond_merged")),
    )


def is_local_macro_host(macro_base: str) -> bool:
    host = (urlparse(macro_base).hostname or "").lower()
    return host in ("127.0.0.1", "localhost", "::1")


def guard_apply_target(*, apply: bool, macro_base: str, allow_production_host: bool) -> tuple[bool, str]:
    if not apply:
        return True, ""
    if allow_production_host:
        return True, ""
    if is_local_macro_host(macro_base):
        return True, ""
    host = urlparse(macro_base).hostname or macro_base
    return False, (
        f"refusing --apply to non-local macro-api host {host!r}; "
        "use --allow-production-host only after explicit ops approval"
    )


def require_confirm_no_active_batch(confirm_flag: bool) -> tuple[bool, str]:
    if not confirm_flag:
        return (
            False,
            "must pass --confirm-no-active-batch (attest no in-flight macro batch for --date)",
        )
    return True, ""


def list_active_batch_runs(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
) -> tuple[list[dict[str, Any]], str]:
    headers = {"Authorization": f"Bearer {macro_key}"}
    q = f"date_from={batch_date}&date_to={batch_date}&limit=200"
    status, data, raw = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/batch/runs?{q}",
        headers,
        timeout=90,
    )
    if status != 200:
        return [], f"batch/runs list HTTP {status}: {raw[:200]}"
    block = data.get("data") if isinstance(data, dict) else data
    items = block.get("items") if isinstance(block, dict) else []
    if not isinstance(items, list):
        return [], "batch/runs items invalid"
    active: list[dict[str, Any]] = []
    for row in items:
        if not isinstance(row, dict):
            continue
        fin = str(row.get("finished_at") or "").strip()
        if not fin:
            active.append(row)
    return active, ""


def record_failure(state: BackfillState, *, phase: str, key: str, detail: str) -> None:
    state.failures.append({"phase": phase, "key": key, "detail": detail})


def pool_manifest_symbols_from_export(export: dict[str, Any], pool_workflow: str) -> list[str]:
    """Merged pool manifest from audit export (`pools[].symbols` + success items), not monitor hardcode."""
    run = (export.get("pool_runs_by_workflow") or {}).get(pool_workflow)
    if not isinstance(run, dict):
        return []
    seen: set[str] = set()
    ordered: list[str] = []
    for pool in run.get("pools") or []:
        if not isinstance(pool, dict):
            continue
        for s in pool.get("symbols") or []:
            u = str(s or "").strip().upper()
            if u and u not in seen:
                seen.add(u)
                ordered.append(u)
        for it in pool.get("items") or []:
            if not isinstance(it, dict) or not it.get("success"):
                continue
            u = str(it.get("symbol") or "").strip().upper()
            if u and u not in seen:
                seen.add(u)
                ordered.append(u)
    return ordered


def expected_symbols_for_pool_backfill(
    export: dict[str, Any],
    pool_workflow: str,
    items: list[dict[str, Any]],
) -> list[str]:
    manifest = pool_manifest_symbols_from_export(export, pool_workflow)
    if manifest:
        return manifest
    return sorted(
        {
            str(it.get("symbol") or "").strip().upper()
            for it in items
            if str(it.get("symbol") or "").strip()
        }
    )


def iter_importable_symbols(
    export: dict[str, Any],
    report: DayAuditReport,
) -> list[tuple[str, dict[str, Any], dict[str, Any]]]:
    """(pool_workflow, run_doc, item) for each importable symbol."""
    runs = export.get("pool_runs_by_workflow") or {}
    by_key = {(s.pool_workflow, s.symbol.upper()): s for s in report.symbol_rows if s.importable}
    out: list[tuple[str, dict[str, Any], dict[str, Any]]] = []
    for wf, run in runs.items():
        if not isinstance(run, dict):
            continue
        for pool in run.get("pools") or []:
            for it in pool.get("items") or []:
                if not isinstance(it, dict):
                    continue
                sym = str(it.get("symbol") or "").strip().upper()
                if (wf, sym) not in by_key:
                    continue
                out.append((wf, run, it))
    return out


def load_recorded_synthesis(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"{path}: synthesis fixture must be object")
    data.setdefault("ok", True)
    return data