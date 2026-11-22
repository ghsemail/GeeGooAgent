#!/usr/bin/env python3
"""Phase B/C: pool + overall resynthesis backfill (default dry-run)."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any

_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))

from macro_jev_backfill_common import (
    audit_export,
    expected_symbols_for_pool_backfill,
    export_fingerprint,
    guard_apply_target,
    list_active_batch_runs,
    load_backfill_state,
    load_recorded_synthesis,
    parse_export_file,
    pool_manifest_symbols_from_export,
    record_failure,
    require_confirm_no_active_batch,
    save_backfill_state,
    validate_export_dates,
)
from macro_jev_orchestration import (
    load_pool_ledger,
    overall_inputs_from_ledger,
    pool_synth_rows_from_ledger,
    record_pool_ledger,
    synthesis_pool_jev_valid,
)
from macro_jev_results_client import (
    JEV_PIPELINE_WORKFLOWS,
    MacroJevResultsWriter,
    fetch_canonical_symbol_result_ids,
    put_overall_result,
)

from run_macro_monitor_lab_batch import (
    MIN_JEV_REASON_LEN,
    WORKFLOW_MONITOR_POOL,
    http_json,
    load_geegoo_env,
    post_batch_run,
    rfc3339_now,
    synthesize_jev_with_retries,
    synthesize_pool_jev_skill,
)

def _symbol_items_from_state(
    export: dict[str, Any],
    batch_date: str,
    wf: str,
    state_symbols: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    """Rebuild item dicts from export for pool wf, restricted to symbols frozen in phase A state."""
    allowed = {
        k.split(":")[1].upper()
        for k in state_symbols
        if k.startswith(f"{wf}:") and state_symbols[k].get("status") == "success"
    }
    runs = export.get("pool_runs_by_workflow") or {}
    run = runs.get(wf)
    if not isinstance(run, dict):
        return []
    items: list[dict[str, Any]] = []
    for pool in run.get("pools") or []:
        for it in pool.get("items") or []:
            sym = str(it.get("symbol") or "").strip().upper()
            if sym in allowed:
                items.append(dict(it))
    return items


def _resolve_synthesis(
    *,
    agent_base: str,
    label: str,
    rows: list[dict[str, Any]],
    recorded_dir: Path | None,
    pool_workflow: str,
    batch_date: str,
    export: dict[str, Any],
    expected_symbols: list[str] | None = None,
) -> dict[str, Any]:
    if recorded_dir:
        path = recorded_dir / f"{pool_workflow}.json"
        if not path.is_file() and pool_workflow == "overall":
            path = recorded_dir / "overall.json"
        if path.is_file():
            syn = load_recorded_synthesis(path)
            syn.setdefault("session_id", syn.get("agent_session_id", ""))
            syn.setdefault("run_id", syn.get("agent_run_id", ""))
            return syn
    runtime_key, _mcp_token = load_geegoo_env()
    if pool_workflow == "overall":
        return synthesize_jev_with_retries(agent_base, runtime_key, _mcp_token, label, rows)
    sbid = str(export.get("source_batch_id") or export.get("batch_run_id") or "").strip()
    if not sbid:
        return {"ok": False, "error": "source_batch_id missing for pool skill backfill"}
    bd = (batch_date or str(export.get("date") or "")).strip()
    exp = expected_symbols or [str(r.get("symbol") or "").strip() for r in rows if r.get("symbol")]
    return synthesize_pool_jev_skill(
        agent_base,
        runtime_key,
        batch_date=bd,
        pool_workflow=pool_workflow,
        source_batch_id=sbid,
        title=label,
        expected_symbols=exp,
        rows=rows,
    )


def run_one_pool(
    *,
    wf: str,
    batch_date: str,
    export: dict[str, Any],
    state: Any,
    macro_base: str,
    macro_key: str,
    agent_base: str,
    apply: bool,
    recorded_dir: Path | None,
) -> int:
    bstate = state.phase_b.get(wf) or {}
    if bstate.get("status") == "success" and bstate.get("result_id"):
        print(f"  skip pool {wf} (state success) result_id={bstate['result_id']}")
        return 0

    items = _symbol_items_from_state(export, batch_date, wf, state.phase_a)
    if not items:
        print(f"  pool {wf}: no phase A frozen symbols", file=sys.stderr)
        return 2

    label = WORKFLOW_MONITOR_POOL.get(wf, wf)
    sym_ok = all(it.get("success") for it in items)
    expected = expected_symbols_for_pool_backfill(export, wf, items)
    if wf == "bond":
        manifest = pool_manifest_symbols_from_export(export, wf)
        frozen = {str(it.get("symbol") or "").strip().upper() for it in items}
        if manifest:
            missing = sorted(set(manifest) - frozen)
            if missing:
                print(
                    f"  pool bond: export manifest missing frozen phase-A ids for {missing}",
                    file=sys.stderr,
                )
                return 2

    print(
        f"  pool {wf}: synth rows={len(items)} manifest={len(expected)} apply={apply}"
    )
    if not apply:
        return 0

    syn = _resolve_synthesis(
        agent_base=agent_base,
        label=label,
        rows=items,
        recorded_dir=recorded_dir,
        pool_workflow=wf,
        batch_date=batch_date,
        export=export,
        expected_symbols=expected,
    )
    started = rfc3339_now()
    finished = rfc3339_now()
    valid, val_err = synthesis_pool_jev_valid(syn, min_reason_len=MIN_JEV_REASON_LEN)
    pool_jev: dict[str, Any]
    if valid:
        pool_jev = {
            "success": True,
            "jev_direction": syn.get("jev_direction"),
            "jev_reason": syn.get("jev_reason"),
            "confidence": syn.get("confidence"),
            "choices": syn.get("choices"),
            "detail_ref": syn.get("detail_ref") or "diagnose_macro_pool",
            "agent_session_id": syn.get("session_id"),
            "agent_run_id": syn.get("run_id"),
            "finished_at": finished,
            "attempt": syn.get("attempt"),
        }
    else:
        pool_jev = {
            "success": False,
            "error": val_err or str(syn.get("error") or "synthesis invalid"),
            "agent_session_id": syn.get("session_id"),
            "finished_at": finished,
        }

    body = {
        "workflow": wf,
        "date": batch_date,
        "started_at": started,
        "finished_at": finished,
        "success": bool(pool_jev.get("success")) and sym_ok,
        "summary": f"backfill pool {wf}",
        "pools": [{"name": label, "symbols": expected, "items": items}],
        "pool_jev": pool_jev,
    }
    pool_run_id = post_batch_run(macro_base, macro_key, body)
    if not pool_run_id:
        record_failure(state, phase="B", key=wf, detail="batch/runs id missing")
        save_backfill_state(state)
        return 2

    writer = MacroJevResultsWriter(
        macro_base, macro_key, batch_date, wf, pool_run_id, started, http_json
    )
    writer.bind_batch_run(pool_run_id)
    canon, c_err = fetch_canonical_symbol_result_ids(
        macro_base, macro_key, http_json, batch_date, wf
    )
    if c_err:
        print(f"  canonical symbols: {c_err}", flush=True)
    for k, v in state.phase_a.items():
        if not k.startswith(f"{wf}:") or v.get("status") != "success":
            continue
        rid = str(v.get("result_id") or "").strip()
        sym = str(v.get("symbol") or "").strip().upper()
        if rid and sym:
            writer.symbol_result_ids[sym] = rid
    writer.merge_symbol_result_ids(canon)

    out = writer.put_pool_from_synthesis(
        pool_jev if pool_jev.get("success") else pool_jev,
        symbol_items=items,
        all_symbols_ok=sym_ok,
        completed_at=finished,
        extra_symbol_result_ids=canon,
        expected_symbols=expected if wf == "bond" and expected else None,
    )
    st = {
        "status": "success" if (out and out.ok) else "failed",
        "result_id": (out.result_id if out else ""),
        "source_batch_id": pool_run_id,
        "batch_started_at": started,
        "reason": (out.reason if out else "") or (out.error if out else val_err),
    }
    state.phase_b[wf] = st
    save_backfill_state(state)
    if writer.pool_result_id and out and out.ok:
        record_pool_ledger(
            batch_date,
            wf,
            result_id=writer.pool_result_id,
            source_batch_id=pool_run_id,
            batch_started_at=started,
            status="success" if sym_ok else "failed",
            cron_workflow=f"backfill:{wf}",
        )
    return 0 if out and out.ok else 1


def run_overall(
    *,
    batch_date: str,
    macro_base: str,
    macro_key: str,
    agent_base: str,
    apply: bool,
    recorded_dir: Path | None,
    state: Any,
) -> int:
    if state.phase_c.get("status") == "success" and state.phase_c.get("result_id"):
        print(f"  skip overall (state) result_id={state.phase_c['result_id']}")
        return 0

    ledger = load_pool_ledger(batch_date)
    missing = [w for w in JEV_PIPELINE_WORKFLOWS if w not in ledger]
    if missing:
        print(f"overall blocked: ledger missing pools {missing}", file=sys.stderr)
        return 2

    pool_refs, ref_err = overall_inputs_from_ledger(
        macro_base, macro_key, http_json, batch_date, ledger
    )
    synth_rows, syn_err = pool_synth_rows_from_ledger(
        macro_base, macro_key, http_json, batch_date, ledger
    )
    print(f"  overall ledger_ok={bool(pool_refs)} apply={apply}")
    if not apply:
        return 0 if pool_refs and synth_rows else 2
    if not pool_refs or ref_err:
        print(ref_err or "ledger incomplete", file=sys.stderr)
        return 2
    if not synth_rows or syn_err:
        print(syn_err or "synth rows missing", file=sys.stderr)
        return 2

    syn = _resolve_synthesis(
        agent_base=agent_base,
        label="宏观整体",
        rows=synth_rows,
        recorded_dir=recorded_dir,
        pool_workflow="overall",
        batch_date=batch_date,
        export={},
    )
    started = rfc3339_now()
    finished = rfc3339_now()
    valid, val_err = synthesis_pool_jev_valid(syn, min_reason_len=MIN_JEV_REASON_LEN)
    pool_jev: dict[str, Any]
    if valid:
        pool_jev = {
            "success": True,
            "jev_direction": syn.get("jev_direction"),
            "jev_reason": syn.get("jev_reason"),
            "agent_session_id": syn.get("session_id"),
            "agent_run_id": syn.get("run_id"),
            "finished_at": finished,
            "attempt": syn.get("attempt"),
        }
    else:
        pool_jev = {
            "success": False,
            "error": val_err or str(syn.get("error") or "overall synthesis failed"),
            "agent_session_id": syn.get("session_id"),
        }

    overall_run_id = post_batch_run(
        macro_base,
        macro_key,
        {
            "workflow": "overall",
            "date": batch_date,
            "started_at": started,
            "finished_at": finished,
            "success": bool(pool_jev.get("success")),
            "summary": "backfill overall",
            "pools": [
                {
                    "name": "宏观整体",
                    "symbols": [],
                    "items": [{"symbol": "OVERALL.IT", "success": False, "error": "backfill placeholder"}],
                }
            ],
            "pool_jev": pool_jev,
        },
    )
    if not overall_run_id:
        record_failure(state, phase="C", key="overall", detail="batch/runs id missing")
        save_backfill_state(state)
        return 2

    out = put_overall_result(
        macro_base,
        macro_key,
        http_json,
        batch_date=batch_date,
        source_batch_id=overall_run_id,
        batch_started_at=started,
        completed_at=finished,
        pool_jev=pool_jev,
        pool_inputs=pool_refs,
    )
    state.phase_c = {
        "status": "success" if out.ok else "failed",
        "result_id": out.result_id,
        "source_batch_id": overall_run_id,
        "reason": out.reason or out.error,
    }
    save_backfill_state(state)
    return 0 if out.ok else 1


def main() -> int:
    p = argparse.ArgumentParser(description="Phase B/C backfill: pool + overall synthesis")
    p.add_argument("--date", required=True)
    p.add_argument("--from-export", required=True, type=Path)
    p.add_argument(
        "--phase",
        choices=("pools", "overall", "all"),
        default="all",
    )
    p.add_argument("--workflows", default="", help="Comma subset of pool workflows for phase B")
    p.add_argument("--macro-base", default=os.environ.get("MACRO_API_BASE", "http://127.0.0.1:3220"))
    p.add_argument(
        "--macro-key",
        default=os.environ.get("MACRO_API_KEY", os.environ.get("GEEGOO_SIGNAL_MACRO_API_KEY", "")),
    )
    p.add_argument("--agent-base", default=os.environ.get("GEEGOO_AGENT_OP_HTTP_URL", "http://127.0.0.1:8080"))
    p.add_argument("--apply", action="store_true")
    p.add_argument("--confirm-no-active-batch", action="store_true", required=True)
    p.add_argument("--allow-production-host", action="store_true")
    p.add_argument(
        "--recorded-synthesis-dir",
        type=Path,
        default=None,
        help="Replay recorded synthesis JSON (integration/ops only; not fabricated fields)",
    )
    args = p.parse_args()

    ok, err = require_confirm_no_active_batch(bool(args.confirm_no_active_batch))
    if not ok:
        print(err, file=sys.stderr)
        return 2
    ok, err = guard_apply_target(
        apply=bool(args.apply),
        macro_base=args.macro_base.strip(),
        allow_production_host=bool(args.allow_production_host),
    )
    if not ok:
        print(err, file=sys.stderr)
        return 2
    if not args.macro_key.strip():
        print("MACRO_API_KEY required", file=sys.stderr)
        return 2

    batch_date = args.date.strip()
    export = parse_export_file(args.from_export)
    errs = validate_export_dates(export, batch_date)
    if errs:
        for e in errs:
            print(f"export invalid: {e}", file=sys.stderr)
        return 2

    report = audit_export(export, batch_date)
    active, aerr = list_active_batch_runs(
        args.macro_base.strip(), args.macro_key.strip(), http_json, batch_date
    )
    if aerr:
        print(aerr, file=sys.stderr)
        return 2
    if active:
        print(f"active batches: {len(active)}", file=sys.stderr)
        return 2

    state = load_backfill_state(batch_date)
    fp = export_fingerprint(export)
    if state.export_fingerprint and state.export_fingerprint != fp:
        print("export_fingerprint mismatch vs state", file=sys.stderr)
        return 2

    ok_a = sum(1 for v in state.phase_a.values() if v.get("status") == "success")
    print(
        f"phase B/C {'APPLY' if args.apply else 'DRY-RUN'} date={batch_date} "
        f"phase_a_frozen={ok_a} minimal_loop={report.minimal_loop_feasible}"
    )
    if ok_a == 0 and args.apply:
        print("phase A state empty; run macro_jev_backfill_symbols --apply first", file=sys.stderr)
        return 2

    wfs = list(JEV_PIPELINE_WORKFLOWS)
    if args.workflows.strip():
        wfs = [w.strip() for w in args.workflows.split(",") if w.strip()]

    rc = 0
    if args.phase in ("pools", "all"):
        for wf in wfs:
            rc = max(rc, run_one_pool(
                wf=wf,
                batch_date=batch_date,
                export=export,
                state=state,
                macro_base=args.macro_base.strip(),
                macro_key=args.macro_key.strip(),
                agent_base=args.agent_base.strip(),
                apply=bool(args.apply),
                recorded_dir=args.recorded_synthesis_dir,
            ))
    if args.phase in ("overall", "all"):
        rc = max(
            rc,
            run_overall(
                batch_date=batch_date,
                macro_base=args.macro_base.strip(),
                macro_key=args.macro_key.strip(),
                agent_base=args.agent_base.strip(),
                apply=bool(args.apply),
                recorded_dir=args.recorded_synthesis_dir,
                state=state,
            ),
        )
    return rc


if __name__ == "__main__":
    raise SystemExit(main())