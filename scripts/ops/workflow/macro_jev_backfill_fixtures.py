#!/usr/bin/env python3
"""Synthetic audit exports for tests only (not production manifests)."""

from __future__ import annotations

from typing import Any

from macro_jev_results_client import JEV_PIPELINE_WORKFLOWS

_PROD_SHAPE_COUNTS = {
    "cn_a": 3,
    "hk": 3,
    "us": 14,
    "commodity_fx": 4,
    "bond": 5,
    "crypto": 2,
    "volatility": 2,
}

_BOND_FIVE = ("HYG.US", "TLT.US", "511260.SH", "511010.SH", "019547.SH")


def _audit_item(symbol: str, *, session_suffix: str) -> dict[str, Any]:
    return {
        "symbol": symbol,
        "success": True,
        "jev_direction": "long",
        "jev_reason": "production-shape audit symbol reason text ok",
        "detail_ref": "buy:1",
        "agent_session_id": f"sess-{session_suffix}",
        "agent_run_id": f"run-{session_suffix}",
    }


def _symbols_for_pool(wf: str, n: int) -> list[str]:
    if wf == "bond":
        return list(_BOND_FIVE)
    prefix = {
        "cn_a": ("00000", ".SH"),
        "hk": ("80", ".HK"),
        "us": ("SYM", ".US"),
        "commodity_fx": ("CMD", ".US"),
        "crypto": ("CRY", ""),
        "volatility": ("VOL", ""),
    }
    p, suf = prefix.get(wf, ("X", ".US"))
    out: list[str] = []
    for i in range(n):
        if wf == "crypto":
            out.append(f"BTC{i}.CRYPTO")
        elif wf == "volatility":
            out.append(f"VIX{i}")
        else:
            out.append(f"{p}{i}{suf}")
    return out


def build_prod_shape_audit_export(batch_date: str) -> dict[str, Any]:
    """33 success symbols: 3/3/14/4/5/2/2 with bond merged 5-member manifest (incl. .SH)."""
    pool_runs: dict[str, Any] = {}
    idx = 0
    for wf in JEV_PIPELINE_WORKFLOWS:
        n = _PROD_SHAPE_COUNTS[wf]
        syms = _symbols_for_pool(wf, n)
        items = [_audit_item(s, session_suffix=f"{wf}-{idx + i}") for i, s in enumerate(syms)]
        idx += n
        pool_jev: dict[str, Any] | None = None
        if wf == "bond":
            pool_jev = {
                "success": True,
                "jev_direction": "neutral",
                "jev_reason": "legacy bond pool_jev without agent_run_id",
            }
        pool_runs[wf] = {
            "id": f"audit-prodshape-{wf}",
            "date": batch_date,
            "started_at": f"{batch_date}T07:{10 + list(JEV_PIPELINE_WORKFLOWS).index(wf):02d}:00+08:00",
            "finished_at": f"{batch_date}T08:{10 + list(JEV_PIPELINE_WORKFLOWS).index(wf):02d}:00+08:00",
            "pools": [
                {
                    "name": wf,
                    "symbols": syms,
                    "items": items,
                }
            ],
            **({"pool_jev": pool_jev} if pool_jev else {}),
        }
    return {
        "batch_date": batch_date,
        "bond_merged": True,
        "overall_runs": [],
        "pool_runs_by_workflow": pool_runs,
    }