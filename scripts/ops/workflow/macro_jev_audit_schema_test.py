#!/usr/bin/env python3
"""Tests for 2026-11-17–style production audit classification (no network)."""

from __future__ import annotations

import unittest

from macro_jev_audit_schema import (
    JEV_PIPELINE_WORKFLOWS,
    classify_day_export,
    classify_batch_run,
)


def _symbol_item(sym: str) -> dict:
    return {
        "symbol": sym,
        "success": True,
        "jev_direction": "long",
        "jev_reason": "审计用标的结论：宏观与技术面一致，风险收益比支持当前方向判断。",
        "detail_ref": "buy:1",
        "agent_session_id": "chat-audit",
        "agent_run_id": "flow-audit",
    }


def _run(
    run_id: str,
    workflow: str,
    date: str,
    *,
    pool_jev_agent: bool = False,
    n_syms: int = 3,
) -> dict:
    syms = [f"SYM{i}.US" for i in range(n_syms)]
    pool_jev = {
        "success": True,
        "jev_direction": "neutral",
        "jev_reason": "池级审计用合成理由：各标的结论聚合后维持中性偏多的宏观池级判断。",
    }
    if pool_jev_agent:
        pool_jev["agent_session_id"] = "chat-pool"
        pool_jev["agent_run_id"] = "flow-pool"
    return {
        "id": run_id,
        "workflow": workflow,
        "date": date,
        "success": True,
        "pool_jev": pool_jev,
        "pools": [
            {
                "name": workflow,
                "symbols": syms,
                "items": [_symbol_item(s) for s in syms],
            }
        ],
    }


class TestAuditSchema1117(unittest.TestCase):
    def test_production_like_pool_jev_missing_run_id(self) -> None:
        """Matches 11/17 audit: symbols importable, pool needs resynthesis."""
        date = "2026-11-17"
        pools = {
            wf: _run(f"run-{wf}", wf, date, pool_jev_agent=False, n_syms=5)
            for wf in JEV_PIPELINE_WORKFLOWS
        }
        rep = classify_day_export(
            batch_date=date,
            pool_runs_by_workflow=pools,
            overall_runs=[],
            bond_merged=True,
        )
        self.assertEqual(rep.pool_runs, 7)
        self.assertTrue(rep.canonical_import_symbols_ok)
        self.assertEqual(rep.overall_finished_count, 0)
        self.assertTrue(rep.minimal_loop_feasible)
        self.assertIn("pool_jev missing agent identity", " ".join(rep.blockers))
        self.assertIn("phase_a_symbol_put_from_batch_audit", rep.recommended_phases)
        self.assertIn("phase_b_pool_resynthesis_only", rep.recommended_phases)
        self.assertIn("phase_c_overall_resynthesis_only", rep.recommended_phases)

    def test_distinct_source_batch_id_per_pool(self) -> None:
        sym, pool = classify_batch_run(
            _run("id-us-only", "us", "2026-11-17"),
            pool_workflow="us",
        )
        self.assertEqual(pool.source_batch_id, "id-us-only")
        self.assertTrue(all(s.source_batch_id == "id-us-only" for s in sym))

    def test_symbol_not_importable_without_run_id(self) -> None:
        run = _run("r1", "hk", "2026-11-17")
        run["pools"][0]["items"][0].pop("agent_run_id")
        sym, _ = classify_batch_run(run, pool_workflow="hk")
        self.assertFalse(sym[0].importable)
        self.assertIn("agent_run_id", sym[0].missing)

    def test_cannot_full_replay_without_symbol_core(self) -> None:
        date = "2026-11-18"
        rep = classify_day_export(
            batch_date=date,
            pool_runs_by_workflow={},
            overall_runs=[],
        )
        self.assertFalse(rep.minimal_loop_feasible)
        self.assertIn("expected 7 pool runs", rep.blockers[0])


if __name__ == "__main__":
    unittest.main()