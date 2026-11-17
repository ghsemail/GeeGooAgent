#!/usr/bin/env python3
"""Read-only classification of macro batch BSON/API exports for canonical backfill planning.

Does not call production PUT. See docs/MACRO_JEV_BACKFILL_MINIMAL_LOOP.md.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

JEV_PIPELINE_WORKFLOWS = (
    "cn_a",
    "hk",
    "us",
    "commodity_fx",
    "bond",
    "crypto",
    "volatility",
)

SYMBOL_CORE_KEYS = (
    "jev_direction",
    "jev_reason",
    "detail_ref",
    "agent_session_id",
    "agent_run_id",
)


@dataclass
class SymbolAuditRow:
    pool_workflow: str
    symbol: str
    source_batch_id: str
    batch_date: str
    success: bool
    missing: list[str] = field(default_factory=list)
    importable: bool = False


@dataclass
class PoolAuditRow:
    pool_workflow: str
    source_batch_id: str
    batch_date: str
    pool_jev_success: bool
    missing: list[str] = field(default_factory=list)
    can_reuse_pool_jev_text: bool = False
    needs_resynthesis: bool = True


@dataclass
class DayAuditReport:
    batch_date: str
    pool_runs: int = 0
    success_symbols: int = 0
    symbol_rows: list[SymbolAuditRow] = field(default_factory=list)
    pool_rows: list[PoolAuditRow] = field(default_factory=list)
    overall_finished_count: int = 0
    bond_merged: bool = False
    canonical_import_symbols_ok: bool = False
    minimal_loop_feasible: bool = False
    blockers: list[str] = field(default_factory=list)
    recommended_phases: list[str] = field(default_factory=list)


def _missing_symbol_core(item: dict[str, Any]) -> list[str]:
    miss: list[str] = []
    if not item.get("success"):
        miss.append("success")
    reason = str(item.get("jev_reason") or "").strip()
    if len(reason) < 20:
        miss.append("jev_reason")
    for k in ("jev_direction", "detail_ref", "agent_session_id", "agent_run_id"):
        if not str(item.get(k) or "").strip():
            miss.append(k)
    return miss


def classify_batch_run(
    run: dict[str, Any],
    *,
    pool_workflow: str,
) -> tuple[list[SymbolAuditRow], PoolAuditRow]:
    """One finished macro_batch_runs document → symbol rows + pool audit."""
    run_id = str(run.get("id") or "").strip()
    batch_date = str(run.get("date") or "").strip()
    pool_jev = run.get("pool_jev") or {}
    pool_ok = bool(pool_jev.get("success"))
    pool_miss: list[str] = []
    if pool_ok:
        if not str(pool_jev.get("jev_direction") or "").strip():
            pool_miss.append("jev_direction")
        if len(str(pool_jev.get("jev_reason") or "").strip()) < 20:
            pool_miss.append("jev_reason")
        if not str(pool_jev.get("agent_run_id") or "").strip():
            pool_miss.append("agent_run_id")
        if not str(pool_jev.get("agent_session_id") or "").strip():
            pool_miss.append("agent_session_id")
    sym_rows: list[SymbolAuditRow] = []
    for pool in run.get("pools") or []:
        for it in pool.get("items") or []:
            if not isinstance(it, dict):
                continue
            sym = str(it.get("symbol") or "").strip()
            if not sym:
                continue
            miss = _missing_symbol_core(it)
            sym_rows.append(
                SymbolAuditRow(
                    pool_workflow=pool_workflow,
                    symbol=sym,
                    source_batch_id=run_id,
                    batch_date=batch_date,
                    success=bool(it.get("success")),
                    missing=miss,
                    importable=bool(it.get("success")) and not miss,
                )
            )
    prow = PoolAuditRow(
        pool_workflow=pool_workflow,
        source_batch_id=run_id,
        batch_date=batch_date,
        pool_jev_success=pool_ok,
        missing=pool_miss,
        can_reuse_pool_jev_text=pool_ok and not pool_miss,
        needs_resynthesis=pool_ok and bool(pool_miss),
    )
    return sym_rows, prow


def classify_day_export(
    *,
    batch_date: str,
    pool_runs_by_workflow: dict[str, dict[str, Any]],
    overall_runs: list[dict[str, Any]],
    bond_merged: bool = False,
) -> DayAuditReport:
    """Classify a read-only export (e.g. 2025-11-17 production audit)."""
    rep = DayAuditReport(batch_date=batch_date, bond_merged=bond_merged)
    rep.overall_finished_count = sum(1 for r in overall_runs if r.get("success") is not None)
    rep.pool_runs = len(pool_runs_by_workflow)

    for wf, run in pool_runs_by_workflow.items():
        sym, pool = classify_batch_run(run, pool_workflow=wf)
        rep.symbol_rows.extend(sym)
        rep.pool_rows.append(pool)

    rep.success_symbols = sum(1 for s in rep.symbol_rows if s.importable)

    if rep.pool_runs != len(JEV_PIPELINE_WORKFLOWS):
        rep.blockers.append(
            f"expected {len(JEV_PIPELINE_WORKFLOWS)} pool runs got {rep.pool_runs}"
        )
    ids = [p.source_batch_id for p in rep.pool_rows if p.source_batch_id]
    if len(ids) != len(set(ids)):
        rep.blockers.append("duplicate source_batch_id across pools")
    if rep.overall_finished_count == 0:
        rep.blockers.append("overall_finished_count=0")

    if any(p.needs_resynthesis for p in rep.pool_rows):
        rep.blockers.append("pool_jev missing agent identity; requires live resynthesis")

    rep.canonical_import_symbols_ok = (
        rep.success_symbols > 0 and rep.success_symbols == sum(1 for s in rep.symbol_rows if s.success)
    )

    if rep.canonical_import_symbols_ok and not any(
        b.startswith("expected") or b.startswith("duplicate") for b in rep.blockers
    ):
        rep.minimal_loop_feasible = True
        rep.recommended_phases = [
            "phase_a_symbol_put_from_batch_audit",
            "phase_b_pool_resynthesis_only",
            "phase_c_overall_resynthesis_only",
        ]
    elif rep.canonical_import_symbols_ok:
        rep.minimal_loop_feasible = True
        rep.recommended_phases = [
            "phase_a_symbol_put_from_batch_audit",
            "phase_b_pool_resynthesis_only",
        ]
        if rep.overall_finished_count == 0:
            rep.recommended_phases.append("phase_c_overall_resynthesis_only")
        rep.blockers.append("partial: overall or pool count preconditions")

    if bond_merged:
        rep.recommended_phases.append("bond_manifest_merged_finalize_run_only")

    return rep