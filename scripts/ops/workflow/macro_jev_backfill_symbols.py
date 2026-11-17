#!/usr/bin/env python3
"""Phase A: audit import symbol canonical PUT (default dry-run)."""

from __future__ import annotations

import argparse
import json
import os
import sys
from dataclasses import asdict
from pathlib import Path
from typing import Any

_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))

from macro_jev_backfill_common import (
    audit_export,
    export_fingerprint,
    guard_apply_target,
    iter_importable_symbols,
    list_active_batch_runs,
    load_backfill_state,
    parse_export_file,
    record_failure,
    require_confirm_no_active_batch,
    save_backfill_state,
    symbol_state_key,
    validate_export_dates,
)
from macro_jev_results_client import MacroJevResultsWriter

try:
    from run_macro_monitor_lab_batch import http_json
except ImportError:
    from macro_jev_signal_integration_test import http_json  # type: ignore


def _plan_line(wf: str, sym: str, run_id: str, started: str, *, apply: bool) -> str:
    mode = "PUT" if apply else "would PUT"
    return f"  [{mode}] {wf} {sym} source_batch_id={run_id} started_at={started}"


def run_phase_a(
    *,
    batch_date: str,
    export_path: Path,
    macro_base: str,
    macro_key: str,
    apply: bool,
    confirm_no_active_batch: bool,
    allow_production_host: bool,
    skip_running: bool,
) -> int:
    ok, err = require_confirm_no_active_batch(confirm_no_active_batch)
    if not ok:
        print(err, file=sys.stderr)
        return 2

    ok, err = guard_apply_target(
        apply=apply, macro_base=macro_base, allow_production_host=allow_production_host
    )
    if not ok:
        print(err, file=sys.stderr)
        return 2

    export = parse_export_file(export_path)
    date_errs = validate_export_dates(export, batch_date)
    if date_errs:
        for e in date_errs:
            print(f"export invalid: {e}", file=sys.stderr)
        return 2

    report = audit_export(export, batch_date)
    if not report.canonical_import_symbols_ok:
        print("audit: no importable success symbols", file=sys.stderr)
        for s in report.symbol_rows:
            if s.success and not s.importable:
                print(f"  {s.symbol} missing {s.missing}", file=sys.stderr)
        return 2

    active, aerr = list_active_batch_runs(macro_base, macro_key, http_json, batch_date)
    if aerr:
        print(f"active-batch check failed: {aerr}", file=sys.stderr)
        return 2
    if active:
        print(f"active batch runs on {batch_date}: {len(active)}", file=sys.stderr)
        for row in active[:10]:
            print(
                f"  id={row.get('id')} workflow={row.get('workflow')} started_at={row.get('started_at')}",
                file=sys.stderr,
            )
        return 2

    fp = export_fingerprint(export)
    state = load_backfill_state(batch_date)
    if state.export_fingerprint and state.export_fingerprint != fp:
        print(
            "state export_fingerprint mismatch; refuse to mix runs "
            "(delete state file or use matching export)",
            file=sys.stderr,
        )
        return 2
    state.export_fingerprint = fp

    rows = iter_importable_symbols(export, report)
    print(f"phase A {'APPLY' if apply else 'DRY-RUN'} date={batch_date} symbols={len(rows)}")

    rc = 0
    writers: dict[tuple[str, str], MacroJevResultsWriter] = {}

    for wf, run, item in rows:
        run_id = str(run.get("id") or run.get("_id") or "").strip()
        started = str(run.get("started_at") or "").strip()
        finished = str(run.get("finished_at") or item.get("finished_at") or started).strip()
        sym = str(item.get("symbol") or "").strip()
        key = symbol_state_key(wf, sym, run_id)
        prev = state.phase_a.get(key) or {}
        if prev.get("status") == "success" and prev.get("result_id"):
            print(f"  skip (state) {key} result_id={prev['result_id']}")
            continue

        print(_plan_line(wf, sym, run_id, started, apply=apply))
        if not apply:
            continue

        wkey = (wf, run_id)
        if wkey not in writers:
            writers[wkey] = MacroJevResultsWriter(
                macro_base,
                macro_key,
                batch_date,
                wf,
                run_id,
                started,
                http_json,
            )
        writer = writers[wkey]
        writer.bind_batch_run(run_id)

        if not skip_running:
            r0 = writer.put_symbol_running(sym)
            if not r0.ok and r0.reason != "immutable_version_conflict":
                record_failure(state, phase="A", key=key, detail=r0.reason or r0.error)
                rc = 1
                save_backfill_state(state)
                continue

        out = writer.put_symbol_from_batch_item(item, completed_at=finished, revision=1)
        st: dict[str, Any] = {
            "pool_workflow": wf,
            "symbol": sym.upper(),
            "source_batch_id": run_id,
            "batch_started_at": started,
            "status": "success" if out.ok else "failed",
            "result_id": out.result_id,
            "reason": out.reason or out.error,
        }
        state.phase_a[key] = st
        save_backfill_state(state)
        if not out.ok:
            rc = 1

    if not apply:
        print(json.dumps({"audit": asdict_report(report), "planned_symbols": len(rows)}, ensure_ascii=False))
    return rc


def asdict_report(report: Any) -> dict[str, Any]:
    return asdict(report)


def main() -> int:
    p = argparse.ArgumentParser(description="Phase A backfill: symbol canonical PUT from audit export")
    p.add_argument("--date", required=True, help="Pipeline date YYYY-MM-DD (must match export)")
    p.add_argument("--from-export", required=True, type=Path, help="Read-only BSON/API export JSON")
    p.add_argument(
        "--macro-base",
        default=os.environ.get("MACRO_API_BASE", "http://127.0.0.1:3220"),
    )
    p.add_argument(
        "--macro-key",
        default=os.environ.get("MACRO_API_KEY", os.environ.get("GEEGOO_SIGNAL_MACRO_API_KEY", "")),
    )
    p.add_argument(
        "--apply",
        action="store_true",
        help="Execute PUT (default dry-run plan only)",
    )
    p.add_argument(
        "--confirm-no-active-batch",
        action="store_true",
        required=True,
        help="Required attest: no in-flight batch on --date",
    )
    p.add_argument("--allow-production-host", action="store_true")
    p.add_argument(
        "--skip-running-revision",
        action="store_true",
        help="Import success@1 only without running@0 (non-default)",
    )
    args = p.parse_args()
    if not args.macro_key.strip():
        print("MACRO_API_KEY required", file=sys.stderr)
        return 2
    return run_phase_a(
        batch_date=args.date.strip(),
        export_path=args.from_export,
        macro_base=args.macro_base.strip(),
        macro_key=args.macro_key.strip(),
        apply=bool(args.apply),
        confirm_no_active_batch=bool(args.confirm_no_active_batch),
        allow_production_host=bool(args.allow_production_host),
        skip_running=bool(args.skip_running_revision),
    )


if __name__ == "__main__":
    raise SystemExit(main())