#!/usr/bin/env python3
"""PUT client for GeeGooSignal canonical macro JEV results (/macro/jev/results/*).

Contract: GeeGooSignal docs/MACRO_JEV_RESULTS_API.md (PR #23).
"""

from __future__ import annotations

import re
import time
import urllib.error
from dataclasses import dataclass, field
from typing import Any, Callable

from macro_jev_synthesis_format import format_pool_jev_reason


HttpJsonFn = Callable[..., tuple[int, Any, str]]

JEV_PIPELINE_WORKFLOWS = (
    "cn_a",
    "hk",
    "us",
    "commodity_fx",
    "bond",
    "crypto",
    "volatility",
)

PUT_ACCEPT_FALSE_REASONS = frozenset(
    {
        "stale_attempt",
        "batch_started_at_mismatch",
        "immutable_version_conflict",
    }
)


def make_source_batch_id(pool_workflow: str, batch_date: str, batch_started_at: str) -> str:
    """Fallback batch id when macro batch/runs did not return run.id (non-prod only)."""
    compact = re.sub(r"[^0-9T]", "", batch_started_at.split("+")[0])[:15]
    safe_wf = re.sub(r"[^a-zA-Z0-9_]", "_", pool_workflow.strip())
    return f"batch-{safe_wf}-{batch_date}-{compact}"


def _safe_id_component(s: str) -> str:
    s = (s or "").strip()
    if not s:
        return "_"
    return re.sub(r"[/\?# :]", "_", s)


def local_symbol_result_id(
    date: str,
    pool_workflow: str,
    symbol: str,
    source_batch_id: str,
    revision: int = 0,
) -> str:
    base = (
        f"sym:{_safe_id_component(date)}:"
        f"{_safe_id_component(pool_workflow)}:"
        f"{_safe_id_component(symbol.upper())}:"
        f"{_safe_id_component(source_batch_id)}"
    )
    if revision > 0:
        return f"{base}@r{revision}"
    return base


@dataclass
class PutOutcome:
    http_status: int
    accepted: bool
    reason: str = ""
    result_id: str = ""
    raw: str = ""
    error: str = ""

    @property
    def ok(self) -> bool:
        return self.http_status == 200 and self.accepted


@dataclass
class MacroJevResultsWriter:
    macro_base: str
    macro_key: str
    batch_date: str
    pool_workflow: str
    source_batch_id: str
    batch_started_at: str
    http_json: HttpJsonFn
    revision: int = 0
    symbol_result_ids: dict[str, str] = field(default_factory=dict)
    pool_result_id: str = ""
    errors: list[str] = field(default_factory=list)
    symbol_revision_final: int = 1
    put_network_retries: int = 3

    def bind_batch_run(self, run_id: str) -> None:
        rid = (run_id or "").strip()
        if rid:
            self.source_batch_id = rid

    def batch_identity_ready(self) -> bool:
        return bool(self.source_batch_id.strip())

    def _require_batch_identity(self, label: str) -> bool:
        if self.batch_identity_ready():
            return True
        self.errors.append(f"{label}: batch_run_id_missing")
        print(f"jev-results PUT {label} blocked: no macro batch run id", flush=True)
        return False

    def _headers(self) -> dict[str, str]:
        return {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.macro_key}",
        }

    def _parse_put_response(self, status: int, data: Any, raw: str) -> PutOutcome:
        if status != 200:
            msg = ""
            if isinstance(data, dict):
                msg = str(data.get("message") or data.get("error") or "")
            return PutOutcome(
                http_status=status,
                accepted=False,
                error=msg or raw[:300] or f"HTTP {status}",
                raw=raw,
            )
        block = data.get("data") if isinstance(data, dict) else None
        if not isinstance(block, dict):
            return PutOutcome(
                http_status=status,
                accepted=False,
                error="missing data envelope",
                raw=raw,
            )
        return PutOutcome(
            http_status=status,
            accepted=bool(block.get("accepted")),
            reason=str(block.get("reason") or ""),
            result_id=str(block.get("result_id") or ""),
            raw=raw,
        )

    def _record_failure(self, label: str, out: PutOutcome) -> None:
        if out.http_status == 200 and not out.accepted:
            detail = out.reason or "accepted:false"
        else:
            detail = out.error or out.reason or "unknown"
        self.errors.append(f"{label}: {detail}")
        print(f"jev-results PUT {label} rejected: {detail}", flush=True)

    def put_symbol(
        self,
        *,
        symbol: str,
        status: str,
        completed_at: str = "",
        jev_direction: str = "",
        jev_reason: str = "",
        confidence: float | None = None,
        agent_session_id: str = "",
        agent_run_id: str = "",
        detail_ref: str = "",
        attempt: int | None = None,
        lab_episode_summary: dict[str, Any] | None = None,
        revision: int | None = None,
    ) -> PutOutcome:
        if not self._require_batch_identity(f"symbol {symbol}"):
            return PutOutcome(http_status=0, accepted=False, error="batch_run_id_missing")
        body: dict[str, Any] = {
            "date": self.batch_date,
            "pool_workflow": self.pool_workflow,
            "symbol": symbol,
            "status": status,
            "source_batch_id": self.source_batch_id,
            "batch_started_at": self.batch_started_at,
            "revision": self.revision if revision is None else revision,
        }
        if jev_direction:
            body["jev_direction"] = jev_direction
        if jev_reason:
            body["jev_reason"] = jev_reason
        if confidence is not None:
            body["confidence"] = confidence
        if agent_session_id:
            body["agent_session_id"] = agent_session_id
        if agent_run_id:
            body["agent_run_id"] = agent_run_id
        if detail_ref:
            body["detail_ref"] = detail_ref
        if completed_at:
            body["completed_at"] = completed_at
        if attempt is not None:
            body["attempt"] = attempt
        if lab_episode_summary:
            body["lab_episode_summary"] = lab_episode_summary

        out = self._http_put_json(
            f"{self.macro_base.rstrip('/')}/macro/jev/results/symbols",
            body,
        )
        sym = symbol.upper()
        rev = body["revision"]
        if out.ok and out.result_id:
            prev = self.symbol_result_ids.get(sym)
            if not prev or rev >= self.symbol_revision_final:
                self.symbol_result_ids[sym] = out.result_id
        elif not out.ok:
            self._record_failure(f"symbol {sym}", out)
        return out

    def _http_put_json(self, url: str, body: dict[str, Any]) -> PutOutcome:
        last: PutOutcome = PutOutcome(http_status=0, accepted=False, error="no attempt")
        for attempt in range(max(1, self.put_network_retries)):
            try:
                status_code, data, raw = self.http_json(
                    "PUT",
                    url,
                    self._headers(),
                    body,
                    timeout=120,
                )
                out = self._parse_put_response(status_code, data, raw)
                if out.http_status == 200:
                    return out
                last = out
            except urllib.error.URLError as exc:
                last = PutOutcome(http_status=0, accepted=False, error=str(exc))
            if attempt + 1 < self.put_network_retries:
                time.sleep(0.2 * (attempt + 1))
        return last

    def put_symbol_running(self, symbol: str) -> PutOutcome:
        return self.put_symbol(
            symbol=symbol,
            status="running",
            revision=0,
        )

    def put_symbol_from_batch_item(
        self,
        item: dict[str, Any],
        *,
        completed_at: str,
        revision: int | None = None,
    ) -> PutOutcome:
        sym = str(item.get("symbol") or "").strip()
        fin_rev = self.symbol_revision_final if revision is None else revision
        if item.get("success"):
            conf = item.get("confidence")
            try:
                conf_f = float(conf) if conf is not None else 0.7
            except (TypeError, ValueError):
                conf_f = 0.7
            lab_summary = item.get("lab_episode_summary")
            lab_arg: dict[str, Any] | None = None
            if isinstance(lab_summary, dict) and lab_summary:
                lab_arg = lab_summary
            return self.put_symbol(
                symbol=sym,
                status="success",
                completed_at=completed_at,
                jev_direction=str(item.get("jev_direction") or ""),
                jev_reason=str(item.get("jev_reason") or ""),
                confidence=conf_f,
                agent_session_id=str(item.get("agent_session_id") or ""),
                agent_run_id=str(item.get("agent_run_id") or ""),
                detail_ref=str(item.get("detail_ref") or ""),
                revision=fin_rev,
                lab_episode_summary=lab_arg,
            )
        err = str(item.get("error") or "diagnose failed")
        reason = err if len(err) >= 8 else f"标的诊断失败: {err}"
        return self.put_symbol(
            symbol=sym,
            status="failed",
            completed_at=completed_at,
            jev_reason=reason[:4000],
            agent_session_id=str(item.get("agent_session_id") or ""),
            agent_run_id=str(item.get("agent_run_id") or ""),
            detail_ref=str(item.get("detail_ref") or ""),
            revision=fin_rev,
        )

    def put_pool(
        self,
        *,
        status: str,
        completed_at: str,
        jev_direction: str = "",
        jev_reason: str = "",
        confidence: float | None = None,
        agent_session_id: str = "",
        agent_run_id: str = "",
        detail_ref: str = "",
        attempt: int | None = None,
        blocked_reason: str = "",
        inputs: list[dict[str, Any]] | None = None,
        revision: int | None = None,
    ) -> PutOutcome:
        if not self._require_batch_identity(f"pool {self.pool_workflow}"):
            return PutOutcome(http_status=0, accepted=False, error="batch_run_id_missing")
        body: dict[str, Any] = {
            "date": self.batch_date,
            "status": status,
            "source_batch_id": self.source_batch_id,
            "batch_started_at": self.batch_started_at,
            "revision": self.revision if revision is None else revision,
            "inputs": inputs or [],
        }
        if jev_direction:
            body["jev_direction"] = jev_direction
        if jev_reason:
            body["jev_reason"] = jev_reason
        if confidence is not None:
            body["confidence"] = confidence
        if agent_session_id:
            body["agent_session_id"] = agent_session_id
        if agent_run_id:
            body["agent_run_id"] = agent_run_id
        if detail_ref:
            body["detail_ref"] = detail_ref
        if completed_at:
            body["completed_at"] = completed_at
        if attempt is not None:
            body["attempt"] = attempt
        if blocked_reason:
            body["blocked_reason"] = blocked_reason
        out = self._http_put_json(
            f"{self.macro_base.rstrip('/')}/macro/jev/results/pools/{self.pool_workflow}",
            body,
        )
        if out.ok and out.result_id:
            self.pool_result_id = out.result_id
        elif not out.ok:
            self._record_failure(f"pool {self.pool_workflow}", out)
        return out

    def merge_symbol_result_ids(self, extra: dict[str, str]) -> None:
        for sym, rid in extra.items():
            if rid:
                self.symbol_result_ids[sym.upper()] = rid

    def put_pool_from_synthesis(
        self,
        pool_jev: dict[str, Any] | None,
        *,
        symbol_items: list[dict[str, Any]],
        all_symbols_ok: bool,
        completed_at: str,
        extra_symbol_result_ids: dict[str, str] | None = None,
        expected_symbols: list[str] | None = None,
    ) -> PutOutcome | None:
        if pool_jev is None and all_symbols_ok and not extra_symbol_result_ids:
            return None
        if extra_symbol_result_ids:
            self.merge_symbol_result_ids(extra_symbol_result_ids)
        success_syms = {
            str(it.get("symbol") or "").strip().upper()
            for it in symbol_items
            if it.get("success")
        }
        inputs = [
            {"result_id": self.symbol_result_ids[sym]}
            for sym in sorted(success_syms)
            if sym in self.symbol_result_ids
        ]
        exp = [s.strip().upper() for s in (expected_symbols or []) if s.strip()]
        if exp and pool_jev and pool_jev.get("success"):
            missing = sorted(set(exp) - success_syms)
            missing_rid = sorted(set(exp) - set(self.symbol_result_ids.keys()))
            if missing or missing_rid or len(inputs) != len(exp):
                return self.put_pool(
                    status="incomplete",
                    completed_at=completed_at,
                    blocked_reason="symbol_tier_incomplete",
                    jev_reason=(
                        f"池 success 阻塞：预期 {len(exp)} 标的，"
                        f"已 success {len(success_syms)}，canonical result_id {len(inputs)}"
                    ),
                    inputs=inputs,
                )
        if pool_jev and pool_jev.get("success"):
            if not all_symbols_ok or len(inputs) == 0:
                return self.put_pool(
                    status="incomplete",
                    completed_at=completed_at,
                    blocked_reason="symbol_tier_incomplete",
                    jev_reason="池级合成完成但标的层未全部 success",
                    inputs=inputs,
                )
            conf = 0.65
            return self.put_pool(
                status="success",
                completed_at=completed_at,
                jev_direction=str(pool_jev.get("jev_direction") or ""),
                jev_reason=format_pool_jev_reason(str(pool_jev.get("jev_reason") or "")),
                confidence=conf,
                agent_session_id=str(pool_jev.get("agent_session_id") or ""),
                agent_run_id=str(pool_jev.get("agent_run_id") or ""),
                attempt=pool_jev.get("attempt"),
                inputs=inputs,
            )
        if not all_symbols_ok:
            blocked = "symbol_tier_incomplete"
            fail_n = sum(1 for it in symbol_items if not it.get("success"))
            return self.put_pool(
                status="failed",
                completed_at=completed_at,
                blocked_reason=blocked,
                jev_reason=f"池级 JEV 阻塞：{fail_n} 个标的未成功",
                inputs=inputs,
            )
        err = str((pool_jev or {}).get("error") or "pool jev failed")
        return self.put_pool(
            status="failed",
            completed_at=completed_at,
            blocked_reason="pool_jev_synthesis_failed",
            jev_reason=err[:4000],
            agent_session_id=str((pool_jev or {}).get("agent_session_id") or ""),
            attempt=(pool_jev or {}).get("attempt"),
            inputs=inputs,
        )

    def has_errors(self) -> bool:
        return len(self.errors) > 0


def fetch_canonical_symbol_result_ids(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
    pool_workflow: str,
) -> tuple[dict[str, str], str]:
    """Latest canonical symbol result_id per symbol for one pool (source=canonical)."""
    headers = {"Authorization": f"Bearer {macro_key}"}
    q = f"date={batch_date}&pool_workflow={pool_workflow}&source=canonical"
    status, data, raw = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/jev/results/symbols?{q}",
        headers,
        timeout=90,
    )
    if status != 200:
        return {}, f"GET symbols HTTP {status}: {raw[:200]}"
    block = data.get("data") if isinstance(data, dict) else {}
    items = block.get("items") if isinstance(block, dict) else []
    if not isinstance(items, list):
        return {}, "invalid symbols list"
    out: dict[str, str] = {}
    for row in items:
        if not isinstance(row, dict):
            continue
        sym = str(row.get("symbol") or "").strip().upper()
        rid = str(row.get("result_id") or "").strip()
        if sym and rid:
            out[sym] = rid
    return out, ""


def get_canonical_pool_result(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
    pool_workflow: str,
) -> tuple[dict[str, Any] | None, str]:
    headers = {"Authorization": f"Bearer {macro_key}"}
    q = f"date={batch_date}&source=canonical"
    status, data, raw = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/jev/results/pools/{pool_workflow}?{q}",
        headers,
        timeout=90,
    )
    if status == 404:
        return None, ""
    if status != 200:
        return None, f"GET pool {pool_workflow} HTTP {status}: {raw[:200]}"
    block = data.get("data") if isinstance(data, dict) else {}
    result = block.get("result") if isinstance(block, dict) else None
    if not isinstance(result, dict):
        return None, f"pool {pool_workflow}: empty canonical result"
    return result, ""


def collect_pool_jev_rows_for_synthesis(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
) -> tuple[list[dict[str, Any]] | None, str]:
    """Read seven pool canonical rows for overall synthesis (direction/reason from truth table)."""
    rows: list[dict[str, Any]] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        canon, err = get_canonical_pool_result(
            macro_base, macro_key, http_json, batch_date, wf
        )
        if err:
            return None, err
        if not canon:
            return None, f"canonical pool missing for {wf}"
        st = str(canon.get("status") or "")
        if st != "success":
            return None, f"pool {wf} canonical status={st}"
        rows.append(
            {
                "label": wf,
                "jev_direction": canon.get("jev_direction"),
                "jev_reason": canon.get("jev_reason"),
            }
        )
    return rows, ""


def collect_overall_pool_inputs(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    batch_date: str,
) -> tuple[list[dict[str, Any]] | None, str]:
    inputs: list[dict[str, Any]] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        row, err = get_canonical_pool_result(
            macro_base, macro_key, http_json, batch_date, wf
        )
        if err:
            return None, err
        if not row:
            return None, f"canonical pool missing for {wf}"
        st = str(row.get("status") or "")
        if st != "success":
            return None, f"pool {wf} canonical status={st}"
        rid = str(row.get("result_id") or "").strip()
        if not rid:
            return None, f"pool {wf} missing result_id"
        inputs.append({"pool_workflow": wf, "result_id": rid})
    return inputs, ""


def put_overall_result(
    macro_base: str,
    macro_key: str,
    http_json: HttpJsonFn,
    *,
    batch_date: str,
    source_batch_id: str,
    batch_started_at: str,
    completed_at: str,
    pool_jev: dict[str, Any],
    pool_inputs: list[dict[str, Any]],
    revision: int = 0,
) -> PutOutcome:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {macro_key}",
    }
    if pool_jev.get("success"):
        body: dict[str, Any] = {
            "date": batch_date,
            "status": "success",
            "jev_direction": pool_jev.get("jev_direction"),
            "jev_reason": pool_jev.get("jev_reason"),
            "confidence": 0.6,
            "agent_session_id": pool_jev.get("agent_session_id"),
            "agent_run_id": pool_jev.get("agent_run_id"),
            "source_batch_id": source_batch_id,
            "batch_started_at": batch_started_at,
            "completed_at": completed_at,
            "revision": revision,
            "inputs": pool_inputs,
            "attempt": pool_jev.get("attempt"),
        }
    else:
        body = {
            "date": batch_date,
            "status": "failed",
            "jev_reason": str(pool_jev.get("error") or "overall jev failed")[:4000],
            "blocked_reason": "overall_jev_synthesis_failed",
            "source_batch_id": source_batch_id,
            "batch_started_at": batch_started_at,
            "completed_at": completed_at,
            "revision": revision,
            "inputs": pool_inputs,
            "agent_session_id": pool_jev.get("agent_session_id"),
        }
    if not source_batch_id.strip():
        return PutOutcome(
            http_status=0,
            accepted=False,
            error="source_batch_id required (bind macro batch run id)",
        )

    status, data, raw = http_json(
        "PUT",
        f"{macro_base.rstrip('/')}/macro/jev/results/overall",
        headers,
        body,
        timeout=120,
    )
    out = PutOutcome(http_status=status, accepted=False, raw=raw)
    if status == 200 and isinstance(data, dict):
        block = data.get("data")
        if isinstance(block, dict):
            out.accepted = bool(block.get("accepted"))
            out.reason = str(block.get("reason") or "")
            out.result_id = str(block.get("result_id") or "")
            if not out.accepted and out.reason:
                out.error = out.reason
    elif status != 200:
        out.error = raw[:300] or f"HTTP {status}"
    return out