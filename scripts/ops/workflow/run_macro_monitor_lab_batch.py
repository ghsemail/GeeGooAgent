#!/usr/bin/env python3
"""宏观监控池日批：逐标的 signal_diagnose → macro inline/ingest + batch/runs.

Chat 与 runtime 之间使用 agent_chat_stream：SSE 分块读、IncompleteRead 后按 session 轮询
``GET /v1/sessions/status``，必要时 ``POST /v1/chat/workflow/resume``。池级失败后可用::

  --pool-jev-only --recover-session-id <chat-id>

仅补写 pool_jev，不重跑标的。池合成在当日 monitor 池标的 canonical 全部 success 后自动触发（合并 batch 行，不仅当次 --symbols）。``overall`` 默认只读 ledger（``--rerun-missing-pools`` 才补跑缺池）。环境变量见 ``agent_chat_stream`` 模块说明。

标的成功门禁与 Signal ``lab_jev_health`` 对齐：``lab-detail`` 四维 lane 健康后才 ``success``/ingest/PUT。
可选：``MACRO_BATCH_CHAT_TIMEOUT``（默认 900，池/overall chat 合成）、``MACRO_BATCH_LAB_DETAIL_TIMEOUT``（默认 900，对齐 agent-runtime 960s 写超时）、``MACRO_BATCH_LAB_DETAIL_HTTP_TRANSIENT_RETRIES``（默认 2）、``MACRO_BATCH_LAB_DETAIL_RETRIES``（默认 2）、``MACRO_BATCH_LAB_DETAIL_RETRY_WAIT``（默认 20）。
"""

from __future__ import annotations

import argparse
import http.client
import json
import os
import re
import sys
import time
import traceback
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))
from agent_chat_stream import (  # noqa: E402
    recover_turn_from_session,
    result_from_stream_error,
    stream_chat_turn,
)
from macro_jev_orchestration import (  # noqa: E402
    POOL_RERUN_CRON_WORKFLOWS,
    load_pool_ledger,
    macro_batch_date_for_cron,
    overall_inputs_from_ledger,
    pool_synth_rows_from_ledger,
    pools_needing_rerun,
    record_pool_ledger,
    resolve_run_dates,
    synthesis_pool_jev_valid,
)
from macro_jev_results_client import (  # noqa: E402
    JEV_PIPELINE_WORKFLOWS,
    MacroJevResultsWriter,
    fetch_canonical_symbol_result_ids,
    put_overall_result,
)
from macro_jev_synthesis_format import format_pool_jev_reason  # noqa: E402
from macro_lab_batch_health import (  # noqa: E402
    build_lab_episode_summary_from_detail,
    direction_from_jev_unified,
    item_refs_complete,
    lab_detail_four_lane_failed,
    lab_episode_summary_failed,
)

DEFAULT_MACRO = "http://146.56.225.252:3220"
DEFAULT_AGENT = "http://127.0.0.1:3400"
WORKFLOW_OPTIONS = {
    "signal_diagnose": {
        "panel_strategy_label": "买:SAR · 卖:SAR",
        "frequency": "60m",
        "months_back": 3,
        "buy_signal": [{"index": "SAR", "type": "signal"}],
        "sell_signal": [{"index": "SAR", "type": "signal"}],
    },
}

# One Lab workflow per monitor-pool tab (+ overall).
WORKFLOW_MONITOR_POOL: dict[str, str] = {
    "cn_a": "A股池",
    "hk": "港股池",
    "us": "美股池",
    "commodity_fx": "商品货币池",
    "bond": "债券利率池",
    "crypto": "虚拟货币池",
    "volatility": "波动率池",
}

JEV_PIPELINE_WORKFLOWS = list(WORKFLOW_MONITOR_POOL.keys())
# Bond split crons (batch/workflow id remains bond).
BOND_SEGMENT_WORKFLOWS = frozenset({"bond_sh", "bond_us"})
BOND_SYMBOL_SUFFIX: dict[str, str] = {"bond_sh": ".SH", "bond_us": ".US"}

# Must exceed server lab-detail wall time (MTF+PA+TA+~4m JEV); agent-runtime WriteTimeout is 960s.
LAB_DETAIL_TIMEOUT = int(os.environ.get("MACRO_BATCH_LAB_DETAIL_TIMEOUT", "900"))
LAB_DETAIL_HTTP_TRANSIENT_RETRIES = int(
    os.environ.get("MACRO_BATCH_LAB_DETAIL_HTTP_TRANSIENT_RETRIES", "2")
)
LAB_DETAIL_RETRIES = int(os.environ.get("MACRO_BATCH_LAB_DETAIL_RETRIES", "2"))
LAB_DETAIL_RETRY_WAIT = int(os.environ.get("MACRO_BATCH_LAB_DETAIL_RETRY_WAIT", "20"))
# Brief pause after chat diagnose before lab-detail (JEV lanes under concurrent morning crons).
LAB_DETAIL_POST_DIAGNOSE_DELAY_SEC = float(
    os.environ.get("MACRO_BATCH_LAB_DETAIL_POST_DIAGNOSE_DELAY_SEC", "8")
)
MIN_JEV_REASON_LEN = 20
MAX_DIAGNOSE_RETRIES = 3
# 单标的/池级合成失败：进程内等待 RETRY_DELAY_SEC（默认 600s = 10min）后重试，最多 MAX_DIAGNOSE_RETRIES 次。
RETRY_DELAY_SEC = int(os.environ.get("MACRO_BATCH_RETRY_DELAY_SEC", "600"))
# Pool/overall synthesis: align run_id backfill with symbol diagnose (12×3s was too short).
POOL_ENSURE_RUN_ID_POLLS = int(os.environ.get("MACRO_BATCH_POOL_RUN_ID_POLLS", "24"))
POOL_ENSURE_RUN_ID_WAIT_SEC = float(os.environ.get("MACRO_BATCH_POOL_RUN_ID_WAIT_SEC", "5.0"))
DIAGNOSING_ERROR = "诊断中"


def load_geegoo_env() -> tuple[str, str]:
    home = os.path.expanduser("~/.geegoo")
    env: dict[str, str] = {}
    path = os.path.join(home, "agent.env")
    if os.path.isfile(path):
        with open(path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                k = k.strip().removeprefix("export ")
                env[k] = v.strip().strip('"').strip("'")
    cfg_path = os.path.join(home, "config.json")
    cfg = json.load(open(cfg_path, encoding="utf-8")) if os.path.isfile(cfg_path) else {}
    runtime_key = (
        os.environ.get("GEEGOO_AGENT_API_KEY", "")
        or env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "")
        or cfg.get("api_key", "")
    ).strip()
    mcp_token = (cfg.get("mcp_token") or "").strip()
    if not runtime_key or not mcp_token:
        raise RuntimeError("need GEEGOO_AGENT_API_KEY + ~/.geegoo/config.json mcp_token")
    return runtime_key, mcp_token


def _http_transient(exc: BaseException) -> bool:
    if isinstance(
        exc,
        (TimeoutError, urllib.error.URLError, http.client.RemoteDisconnected, ConnectionResetError),
    ):
        return True
    return "Remote end closed connection" in str(exc)


def http_json(
    method: str,
    url: str,
    headers: dict[str, str],
    body: dict[str, Any] | None = None,
    timeout: int = 120,
    *,
    transient_retries: int = 0,
) -> tuple[int, Any, str]:
    data = json.dumps(body or {}, ensure_ascii=False).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    attempts = max(0, transient_retries) + 1
    last_msg = "request failed"
    for attempt in range(attempts):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                raw = resp.read().decode("utf-8", errors="replace")
                return resp.status, json.loads(raw) if raw.strip() else {}, raw
        except urllib.error.HTTPError as e:
            raw = e.read().decode("utf-8", errors="replace")
            try:
                parsed = json.loads(raw) if raw.strip() else {}
            except json.JSONDecodeError:
                parsed = {"raw": raw}
            return e.code, parsed, raw
        except TimeoutError:
            last_msg = "timeout"
            if attempt + 1 < attempts:
                time.sleep(min(5 * (attempt + 1), 20))
                continue
            return 504, {"error": "timeout"}, "timeout"
        except urllib.error.URLError as e:
            reason = getattr(e, "reason", e)
            last_msg = str(reason)
            if attempt + 1 < attempts and _http_transient(e):
                time.sleep(min(5 * (attempt + 1), 20))
                continue
            return 0, {"error": last_msg}, last_msg
        except (http.client.RemoteDisconnected, ConnectionResetError) as e:
            last_msg = str(e)
            if attempt + 1 < attempts:
                time.sleep(min(5 * (attempt + 1), 20))
                continue
            return 0, {"error": last_msg}, last_msg
        except OSError as e:
            last_msg = str(e)
            if attempt + 1 < attempts and _http_transient(e):
                time.sleep(min(5 * (attempt + 1), 20))
                continue
            return 0, {"error": last_msg}, last_msg
    return 0, {"error": last_msg}, last_msg


def resolve_batch_workflow(workflow: str) -> str:
    if workflow in BOND_SEGMENT_WORKFLOWS:
        return "bond"
    return workflow


def rfc3339_now() -> str:
    """UTC RFC3339 for macro batch/runs and jev-results (avoids batch_started_at lock skew)."""
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def ensure_diagnose_run_id(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str,
    *,
    polls: int = 24,
    poll_wait_sec: float = 5.0,
) -> str:
    """Backfill run_id after chat/stream when SSE omitted it (pool synthesis validation)."""
    rid = (run_id or "").strip()
    sid = (session_id or "").strip()
    if rid or not sid:
        return rid
    for poll in range(polls):
        res = fetch_diagnose_result(agent_base, runtime_key, mcp_token, sid, "")
        if res.get("ok"):
            data = res["data"]
            if isinstance(data, dict) and isinstance(data.get("data"), dict):
                data = data["data"]
            rid = str(data.get("run_id") or "").strip()
            if rid:
                return rid
        if poll + 1 < polls:
            time.sleep(poll_wait_sec)
    return ""


def fetch_monitor_pool(macro_base: str, macro_key: str) -> list[dict[str, Any]]:
    status, data, _ = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/monitor/pool",
        {"Authorization": f"Bearer {macro_key}"},
        timeout=60,
    )
    if status != 200:
        raise RuntimeError(f"monitor/pool HTTP {status}: {data}")
    items = data.get("data", {}).get("items") or data.get("items") or []
    return list(items)


def group_symbols_for_workflow(pool: list[dict[str, Any]], workflow: str) -> dict[str, list[dict[str, Any]]]:
    pool_name = WORKFLOW_MONITOR_POOL.get(workflow) or (
        WORKFLOW_MONITOR_POOL["bond"] if workflow in BOND_SEGMENT_WORKFLOWS else ""
    )
    if not pool_name:
        return {}
    suffix = BOND_SYMBOL_SUFFIX.get(workflow, "")
    items: list[dict[str, Any]] = []
    for it in pool:
        name = (it.get("monitor_reason") or "").strip()
        if name != pool_name:
            continue
        sym = (it.get("symbol") or "").strip()
        if suffix and not sym.upper().endswith(suffix):
            continue
        if sym:
            items.append(it)
    if not items:
        return {}
    return {pool_name: items}


def diagnose_message(name: str, symbol: str) -> str:
    label = (name or symbol).strip()
    sym = symbol.strip()
    return (
        f"信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 抛物线信号作为卖出信号，"
        f"开启阻力支撑熔断，测试一下{label}（{sym}）。"
    )


def chat_diagnose(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    message: str,
    *,
    session_id: str = "",
) -> dict[str, Any]:
    """POST /v1/chat/stream with SSE recovery via /v1/sessions/status and workflow resume."""
    min_reply = MIN_JEV_REASON_LEN
    turn = stream_chat_turn(
        agent_base,
        runtime_key,
        mcp_token,
        message,
        body_extra={"workflow_options": WORKFLOW_OPTIONS},
        session_id=session_id,
        recover=True,
        min_reply_len=min_reply,
    )
    out = result_from_stream_error(turn)
    if turn.recovered or turn.resumed:
        print(
            f"  chat stream recover: recovered={turn.recovered} resumed={turn.resumed}"
            f" session={out.get('session_id', '')[:20]}",
            flush=True,
        )
    if turn.stream_error and not out.get("ok"):
        print(f"  chat stream error: {turn.stream_error[:280]}", flush=True)
    if out.get("ok") and not out.get("run_id"):
        sid = str(out.get("session_id") or "").strip()
        if sid:
            for poll in range(24):
                res = fetch_diagnose_result(agent_base, runtime_key, mcp_token, sid, "")
                if res.get("ok"):
                    data = res["data"]
                    if isinstance(data, dict) and isinstance(data.get("data"), dict):
                        data = data["data"]
                    rid = str(data.get("run_id") or "").strip()
                    if rid:
                        out["run_id"] = rid
                        break
                if poll < 23:
                    time.sleep(5)
    return out


def pool_jev_from_session(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
) -> dict[str, Any]:
    turn = recover_turn_from_session(
        agent_base,
        runtime_key,
        mcp_token,
        session_id,
        min_reply_len=MIN_JEV_REASON_LEN,
    )
    if not turn.ok:
        return {
            "ok": False,
            "error": turn.stream_error or "recover session failed",
            "session_id": session_id,
        }
    jev_reason = str(turn.reply or "").strip()
    direction = parse_direction_from_compose(jev_reason)
    if len(jev_reason) < MIN_JEV_REASON_LEN or not direction:
        return {
            "ok": False,
            "error": "incomplete pool jev from session",
            "session_id": session_id,
            "jev_reason": jev_reason,
        }
    return {
        "ok": True,
        "jev_direction": direction,
        "jev_reason": jev_reason,
        "session_id": session_id,
        "run_id": turn.run_id,
        "recovered": True,
    }


def fetch_diagnose_result(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str = "",
) -> dict[str, Any]:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-MCP-Token": mcp_token,
    }
    body: dict[str, Any] = {"session_id": session_id, "mcp_token": mcp_token}
    if run_id:
        body["run_id"] = run_id
    status, data, raw = http_json(
        "POST",
        f"{agent_base.rstrip('/')}/v1/signal-diagnose/result",
        headers,
        body,
        timeout=120,
    )
    if status != 200:
        return {"ok": False, "error": f"result HTTP {status}: {raw[:400]}"}
    return {"ok": True, "data": data}


def _episode_ref_sort_key(ref: str) -> int:
    if ":" not in ref:
        return 0
    tail = ref.split(":", 1)[1]
    return int(tail) if tail.isdigit() else 0


def _best_episode_ref(candidates: list[str]) -> str:
    keys = [c.strip() for c in candidates if c.strip().startswith(("buy:", "sell:"))]
    if not keys:
        return ""
    keys.sort(key=_episode_ref_sort_key, reverse=True)
    for pref in ("buy:", "sell:"):
        for k in keys:
            if k.startswith(pref):
                return k
    return keys[0]


def _ref_from_lab_episode_summaries(lab: dict[str, Any]) -> str:
    summaries = lab.get("lab_episode_summaries")
    if not isinstance(summaries, dict):
        return ""
    best_ref = ""
    best_idx = -1
    for key, raw in summaries.items():
        if not isinstance(raw, dict):
            continue
        start_idx = raw.get("start_idx")
        if start_idx is None:
            continue
        try:
            idx = int(start_idx)
        except (TypeError, ValueError):
            continue
        side = ""
        k = str(key).strip().lower()
        if k.startswith("buy:"):
            side = "buy"
        elif k.startswith("sell:"):
            side = "sell"
        if not side:
            continue
        if idx > best_idx:
            best_idx = idx
            best_ref = f"{side}:{idx}"
    return best_ref


def pick_detail_ref(result: dict[str, Any]) -> str:
    lab = result.get("lab") or {}
    ref = (lab.get("lab_default_detail_ref") or "").strip()
    if ref.startswith(("buy:", "sell:")):
        return ref
    from_summaries = _ref_from_lab_episode_summaries(lab)
    if from_summaries:
        return from_summaries
    signal_eval = result.get("signal_eval") or {}
    eps: list[str] = []
    for ep in signal_eval.get("buy_details") or []:
        if isinstance(ep, dict):
            eps.append(_detail_ref_from_signal_eval_episode(ep))
    for ep in signal_eval.get("sell_details") or []:
        if isinstance(ep, dict):
            eps.append(_detail_ref_from_signal_eval_episode(ep))
    return _best_episode_ref(eps)


def _detail_ref_from_signal_eval_episode(ep: dict[str, Any]) -> str:
    ref = str(ep.get("detail_ref") or ep.get("ref") or "").strip()
    if ref.startswith(("buy:", "sell:")):
        return ref
    start_idx = ep.get("start_idx")
    if start_idx is None:
        return ""
    try:
        idx = int(start_idx)
    except (TypeError, ValueError):
        return ""
    side = str(ep.get("side") or "").strip().lower()
    if side not in ("buy", "sell"):
        for key in ("action", "signal_side"):
            side = str(ep.get(key) or "").strip().lower()
            if side in ("buy", "sell"):
                break
    if side not in ("buy", "sell"):
        return ""
    return f"{side}:{idx}"


def is_valid_detail_ref(ref: str) -> bool:
    r = (ref or "").strip()
    return r.startswith("buy:") or r.startswith("sell:")


def item_jev_complete(
    jev_reason: str,
    detail_ref: str,
    session_id: str,
    run_id: str,
    direction: str,
    lab_episode_summary: dict[str, Any] | None = None,
) -> bool:
    ok, _ = item_refs_complete(
        jev_reason, detail_ref, session_id, run_id, direction, min_reason_len=MIN_JEV_REASON_LEN
    )
    if not ok:
        return False
    failed, _ = lab_episode_summary_failed(lab_episode_summary)
    return not failed


def post_batch_run(
    macro_base: str,
    macro_key: str,
    body: dict[str, Any],
) -> str:
    macro_headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {macro_key}",
    }
    try:
        status, data, raw = http_json(
            "POST",
            f"{macro_base.rstrip('/')}/macro/batch/runs",
            macro_headers,
            body,
            timeout=120,
        )
    except urllib.error.URLError as e:
        print(f"batch/runs transport error: {e}", file=sys.stderr, flush=True)
        return ""
    if status != 200:
        print(f"batch/runs HTTP {status}: {raw[:400]}", file=sys.stderr, flush=True)
        return ""
    run_id = (data.get("data") or {}).get("run", {}).get("id") or data.get("id") or ""
    return str(run_id).strip()


def _unwrap_data(payload: Any) -> dict[str, Any]:
    if isinstance(payload, dict) and isinstance(payload.get("data"), dict):
        return payload["data"]
    return payload if isinstance(payload, dict) else {}


def merged_batch_symbol_rows(
    macro_base: str,
    macro_key: str,
    workflow: str,
    batch_date: str,
) -> list[dict[str, Any]]:
    headers = {"Authorization": f"Bearer {macro_key}"}
    q = f"workflow={workflow}&date_from={batch_date}&date_to={batch_date}&limit=40"
    status, data, _ = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/batch/runs?{q}",
        headers,
        timeout=90,
    )
    if status != 200:
        return []
    items = _unwrap_data(data).get("items") or []
    by_sym: dict[str, dict[str, Any]] = {}
    for row in reversed(items):
        rid = str(row.get("id") or "").strip()
        if not rid:
            continue
        st, detail, _ = http_json(
            "GET",
            f"{macro_base.rstrip('/')}/macro/batch/runs/{rid}",
            headers,
            timeout=90,
        )
        if st != 200:
            continue
        run = _unwrap_data(detail).get("run") or {}
        for pool in run.get("pools") or []:
            for it in pool.get("items") or []:
                sym = str(it.get("symbol") or "").strip()
                if not sym or not it.get("success"):
                    continue
                if not str(it.get("jev_direction") or "").strip():
                    continue
                by_sym[sym] = it
    return list(by_sym.values())


def monitor_pool_symbols(grouped: dict[str, list[dict[str, Any]]]) -> list[str]:
    syms: set[str] = set()
    for items in grouped.values():
        for it in items:
            sym = str(it.get("symbol") or "").strip()
            if sym:
                syms.add(sym)
    return sorted(syms)


def resolve_pool_synthesis_rows(
    macro_base: str,
    macro_key: str,
    workflow: str,
    batch_date: str,
    expected_symbols: list[str],
) -> tuple[list[dict[str, Any]], bool, str]:
    """Merged successful batch rows for the full monitor pool, or skip with reason."""
    exp = [s.strip() for s in expected_symbols if s.strip()]
    if not exp:
        return [], False, "no monitor pool symbols"
    merged = merged_batch_symbol_rows(macro_base, macro_key, workflow, batch_date)
    by_sym = {str(r.get("symbol") or "").strip(): r for r in merged}
    missing = [s for s in exp if s not in by_sym]
    if missing:
        return [], False, "canonical missing: " + ",".join(missing)
    rows = [by_sym[s] for s in sorted(exp)]
    return rows, True, ""


def expected_bond_symbols(macro_base: str, macro_key: str) -> list[str]:
    syms = []
    for it in fetch_monitor_pool(macro_base, macro_key):
        if (it.get("monitor_reason") or "").strip() != WORKFLOW_MONITOR_POOL["bond"]:
            continue
        sym = str(it.get("symbol") or "").strip()
        if sym:
            syms.append(sym)
    return sorted(syms)


def merged_bond_segments_for_orchestration(
    macro_base: str,
    macro_key: str,
    orchestration_date: str,
) -> list[dict[str, Any]]:
    us_date = macro_batch_date_for_cron("bond_us", orchestration_date)
    sh_date = macro_batch_date_for_cron("bond_sh", orchestration_date)
    by_sym: dict[str, dict[str, Any]] = {}
    for segment_date in (us_date, sh_date):
        for row in merged_batch_symbol_rows(macro_base, macro_key, "bond", segment_date):
            sym = str(row.get("symbol") or "").strip()
            if sym:
                by_sym[sym] = row
    return list(by_sym.values())


def try_finalize_bond_pool_jev(
    macro_base: str,
    macro_key: str,
    agent_base: str,
    orchestration_date: str,
) -> int:
    expected = expected_bond_symbols(macro_base, macro_key)
    if not expected:
        return 0
    merged = merged_bond_segments_for_orchestration(macro_base, macro_key, orchestration_date)
    have = {str(r.get("symbol") or "").strip() for r in merged}
    if set(expected) - have:
        missing = sorted(set(expected) - have)
        print(f"[bond] pool JEV deferred; missing symbols: {missing}", flush=True)
        return 0
    runtime_key, mcp_token = load_geegoo_env()
    print("[bond] pool JEV synthesize (债券利率池, merged segments) …", flush=True)
    syn = synthesize_jev_with_retries(
        agent_base, runtime_key, mcp_token, WORKFLOW_MONITOR_POOL["bond"], merged
    )
    started = rfc3339_now()
    finished = rfc3339_now()
    pool_jev: dict[str, Any]
    if syn.get("ok"):
        pool_jev = {
            "success": True,
            "jev_direction": syn.get("jev_direction"),
            "jev_reason": syn.get("jev_reason"),
            "agent_session_id": syn.get("session_id"),
            "agent_run_id": syn.get("run_id"),
            "finished_at": finished,
            "attempt": syn.get("attempt"),
        }
        print(f"  pool JEV ok={pool_jev.get('jev_direction')}", flush=True)
    else:
        pool_jev = {
            "success": False,
            "error": str(syn.get("error") or "pool jev failed"),
            "attempt": syn.get("attempt"),
            "finished_at": finished,
        }
        print(f"  pool JEV FAIL: {pool_jev.get('error', '')[:240]}", flush=True)
    batch_date = orchestration_date
    body = {
        "workflow": "bond",
        "date": batch_date,
        "started_at": started,
        "finished_at": finished,
        "success": bool(pool_jev.get("success")),
        "summary": "bond finished (merged segments)",
        "pools": [
            {
                "name": WORKFLOW_MONITOR_POOL["bond"],
                "symbols": expected,
                "items": merged,
            }
        ],
        "pool_jev": pool_jev,
    }
    bond_run_id = post_batch_run(macro_base, macro_key, body)
    writer = MacroJevResultsWriter(
        macro_base,
        macro_key,
        batch_date,
        "bond",
        bond_run_id or "",
        started,
        http_json,
    )
    if bond_run_id:
        writer.bind_batch_run(bond_run_id)
    us_date = macro_batch_date_for_cron("bond_us", orchestration_date)
    sh_date = macro_batch_date_for_cron("bond_sh", orchestration_date)
    canon_syms: dict[str, str] = {}
    for seg_date, label in ((us_date, "bond_us"), (sh_date, "bond_sh"), (batch_date, "bond")):
        part, sym_err = fetch_canonical_symbol_result_ids(
            macro_base, macro_key, http_json, seg_date, "bond"
        )
        if sym_err:
            print(f"[bond] canonical symbols ({label} @ {seg_date}): {sym_err}", flush=True)
        canon_syms.update(part)
    missing = [s for s in expected if s.upper() not in canon_syms]
    for it in merged:
        sym = str(it.get("symbol") or "").strip().upper()
        if sym in missing and it.get("success"):
            writer.put_symbol_from_batch_item(it, completed_at=finished)
    canon_syms_reload: dict[str, str] = {}
    for seg_date in (us_date, sh_date, batch_date):
        part, sym_err2 = fetch_canonical_symbol_result_ids(
            macro_base, macro_key, http_json, seg_date, "bond"
        )
        if sym_err2:
            print(f"[bond] canonical symbols reload @ {seg_date}: {sym_err2}", flush=True)
        canon_syms_reload.update(part)
    canon_syms = canon_syms_reload
    writer.put_pool_from_synthesis(
        pool_jev,
        symbol_items=merged,
        all_symbols_ok=True,
        completed_at=finished,
        extra_symbol_result_ids=canon_syms,
        expected_symbols=expected,
    )
    rc = 0 if pool_jev.get("success") else 1
    if writer.has_errors():
        rc = max(rc, 1)
    if writer.pool_result_id and writer.batch_identity_ready():
        pool_st = "success" if pool_jev.get("success") else "failed"
        record_pool_ledger(
            orchestration_date,
            "bond",
            result_id=writer.pool_result_id,
            source_batch_id=writer.source_batch_id,
            batch_started_at=writer.batch_started_at,
            status=pool_st,
            cron_workflow="bond_finalize",
        )
    return rc


def collect_pool_inputs_from_pipeline(
    macro_base: str,
    macro_key: str,
    batch_date: str,
) -> tuple[list[dict[str, Any]] | None, str]:
    headers = {"Authorization": f"Bearer {macro_key}"}
    status, data, raw = http_json(
        "GET",
        f"{macro_base.rstrip('/')}/macro/jev/pipeline?date={batch_date}",
        headers,
        timeout=90,
    )
    if status != 200:
        return None, f"pipeline HTTP {status}: {raw[:200]}"
    root = _unwrap_data(data)
    pools = root.get("pools") or []
    inputs: list[dict[str, Any]] = []
    for wf in JEV_PIPELINE_WORKFLOWS:
        step = next((p for p in pools if p.get("workflow") == wf), None)
        if not step:
            return None, f"pipeline missing workflow={wf}"
        st = str(step.get("status") or "")
        if st not in ("success", "stale"):
            return None, f"pool {wf} status={st}"
        reason = str(step.get("jev_reason") or "").strip()
        direction = str(step.get("jev_direction") or "").strip()
        if not reason or not direction:
            return None, f"pool {wf} missing jev"
        inputs.append(
            {
                "label": WORKFLOW_MONITOR_POOL[wf],
                "jev_direction": direction,
                "jev_reason": reason,
            }
        )
    return inputs, ""


def diagnose_symbol_once(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    name: str,
    sym: str,
) -> dict[str, Any]:
    msg = diagnose_message(name, sym)
    chat = chat_diagnose(agent_base, runtime_key, mcp_token, msg)
    if not chat.get("ok"):
        return {"ok": False, "error": chat.get("error") or "chat failed"}

    session_id = str(chat.get("session_id") or "").strip()
    hint_run = str(chat.get("run_id") or "").strip()
    res = fetch_diagnose_result(agent_base, runtime_key, mcp_token, session_id, hint_run)
    if not res.get("ok"):
        return {"ok": False, "error": res.get("error") or "result failed"}

    data = res["data"]
    if isinstance(data, dict) and isinstance(data.get("data"), dict):
        data = data["data"]
    run_id = str(data.get("run_id") or hint_run or "").strip()
    jev_reason = str(data.get("eval_judgment") or "").strip()
    if len(jev_reason) < MIN_JEV_REASON_LEN:
        jev_reason = str(chat.get("reply") or "").strip() or jev_reason
    detail_ref = pick_detail_ref(data)
    lab_summary: dict[str, Any] | None = None
    direction, conf = "", None
    if session_id and run_id and detail_ref:
        if LAB_DETAIL_POST_DIAGNOSE_DELAY_SEC > 0:
            time.sleep(LAB_DETAIL_POST_DIAGNOSE_DELAY_SEC)
        lab = lab_detail_symbol_gate(
            agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref
        )
        if not lab.get("ok"):
            return {
                "ok": False,
                "error": str(lab.get("error") or "lab-detail failed"),
                "session_id": session_id,
                "run_id": run_id,
                "jev_reason": jev_reason,
                "detail_ref": detail_ref,
            }
        lab_summary = lab.get("lab_episode_summary")
        direction = str(lab.get("direction") or "")
        conf = lab.get("confidence")
    if not direction:
        direction = parse_direction_from_compose(jev_reason)
    if conf is None:
        conf = 1.0 if direction != "neutral" else 0.5

    if not item_jev_complete(
        jev_reason, detail_ref, session_id, run_id, direction, lab_summary
    ):
        refs_ok, refs_msg = item_refs_complete(
            jev_reason, detail_ref, session_id, run_id, direction, min_reason_len=MIN_JEV_REASON_LEN
        )
        failed, sum_msg = lab_episode_summary_failed(lab_summary)
        parts: list[str] = []
        if not refs_ok:
            parts.append(refs_msg)
        if failed:
            parts.append(sum_msg)
        return {
            "ok": False,
            "error": " · ".join(parts) or "incomplete jev",
            "session_id": session_id,
            "run_id": run_id,
            "jev_reason": jev_reason,
            "detail_ref": detail_ref,
            "direction": direction,
        }

    return {
        "ok": True,
        "session_id": session_id,
        "run_id": run_id,
        "jev_reason": jev_reason,
        "detail_ref": detail_ref,
        "direction": direction,
        "confidence": conf,
        "lab_episode_summary": lab_summary,
    }


def diagnose_symbol_with_retries(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    name: str,
    sym: str,
) -> dict[str, Any]:
    last: dict[str, Any] = {"ok": False, "error": "no attempt"}
    for attempt in range(MAX_DIAGNOSE_RETRIES + 1):
        if attempt > 0:
            print(f"  retry {attempt}/{MAX_DIAGNOSE_RETRIES} after {RETRY_DELAY_SEC}s …", flush=True)
            time.sleep(RETRY_DELAY_SEC)
        last = diagnose_symbol_once(agent_base, runtime_key, mcp_token, name, sym)
        if last.get("ok"):
            return last
        err = str(last.get("error") or "failed")
        print(f"  attempt {attempt + 1} incomplete: {err[:240]}", flush=True)
    return last


def _post_lab_detail(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str,
    detail_ref: str,
) -> tuple[int, dict[str, Any], str]:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-MCP-Token": mcp_token,
    }
    body = {
        "session_id": session_id,
        "run_id": run_id,
        "detail_ref": detail_ref,
        "mcp_token": mcp_token,
        "rebuild": True,
        "kline_jev_profile": "daily_macro",
    }
    status, data, raw = http_json(
        "POST",
        f"{agent_base.rstrip('/')}/v1/signal-diagnose/lab-detail",
        headers,
        body,
        timeout=LAB_DETAIL_TIMEOUT,
        transient_retries=LAB_DETAIL_HTTP_TRANSIENT_RETRIES,
    )
    if not isinstance(data, dict):
        data = {}
    return status, data, raw


def lab_detail_symbol_gate(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str,
    detail_ref: str,
) -> dict[str, Any]:
    """Symbol batch: lab-detail with four-lane health gate (matches Signal pipeline)."""
    last_err = "lab-detail failed"
    for attempt in range(LAB_DETAIL_RETRIES + 1):
        status, data, raw = _post_lab_detail(
            agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref
        )
        if status != 200:
            last_err = f"lab-detail HTTP {status}: {raw[:240]}"
        else:
            detail = data.get("detail") if isinstance(data.get("detail"), dict) else {}
            failed, reason = lab_detail_four_lane_failed(detail)
            summary: dict[str, Any] | None = None
            if not failed:
                summary = build_lab_episode_summary_from_detail(detail, detail_ref)
                if summary:
                    failed, reason = lab_episode_summary_failed(summary)
                else:
                    failed, reason = True, "lab_episode_summary build failed"
            if not failed and summary:
                unified = summary.get("jev_unified")
                if not isinstance(unified, dict):
                    unified = detail.get("jev_unified") if isinstance(detail.get("jev_unified"), dict) else None
                direction, conf = direction_from_jev_unified(unified)
                if direction:
                    return {
                        "ok": True,
                        "direction": direction,
                        "confidence": conf,
                        "lab_episode_summary": summary,
                    }
                last_err = "lab-detail missing jev direction"
            else:
                last_err = reason or "lab_episode_summary invalid"
        if attempt < LAB_DETAIL_RETRIES:
            print(f"  lab-detail retry {attempt + 1}/{LAB_DETAIL_RETRIES}: {last_err[:160]}", flush=True)
            wait = LAB_DETAIL_RETRY_WAIT
            if "lanes missing" in (last_err or ""):
                wait = max(wait, int(os.environ.get("MACRO_BATCH_LAB_LANE_RETRY_WAIT", "45")))
            time.sleep(wait)
    return {"ok": False, "error": last_err}


def lab_detail_jev(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str,
    detail_ref: str,
) -> tuple[str, float | None]:
    """Pool/overall synthesis: direction from jev_unified only (no four-lane gate)."""
    status, data, raw = _post_lab_detail(
        agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref
    )
    if status != 200:
        msg = ""
        if isinstance(data, dict):
            msg = str(data.get("message") or data.get("error") or "").strip()
        if not msg:
            msg = str(raw or "")[:240]
        return "", None
    detail = data.get("detail") if isinstance(data.get("detail"), dict) else {}
    unified = detail.get("jev_unified") or data.get("jev_unified")
    return direction_from_jev_unified(unified if isinstance(unified, dict) else None)


def synthesis_jev_message(title: str, rows: list[dict[str, Any]]) -> str:
    lines: list[str] = []
    for r in rows:
        sym = str(r.get("symbol") or r.get("label") or "").strip()
        direction = str(r.get("jev_direction") or r.get("direction") or "").strip()
        reason = str(r.get("jev_reason") or r.get("reason") or "").strip()
        if len(reason) > 800:
            reason = reason[:800] + "…"
        lines.append(f"- {sym} [{direction}]: {reason}")
    joined = "\n".join(lines)
    return (
        f"宏观 JEV 合成任务：{title}。根据以下各标的或各市场池级 JEV 结论，"
        f"再用 JEV 模型给出整体多空（long/short/neutral）与完整理由，"
        f"禁止简单计数或涨跌幅规则。\n\n"
        "输出要求（必须遵守）：使用简体中文 Markdown；以「## 整体结论」开头，"
        "列出方向、置信度、2-4 条核心依据；正文禁止英文 chain-of-thought，"
        "禁止逐字复述输入。\n\n"
        f"{joined}"
    )


def synthesize_jev_with_retries(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    title: str,
    rows: list[dict[str, Any]],
) -> dict[str, Any]:
    if not rows:
        return {"ok": False, "error": "no inputs"}
    msg = synthesis_jev_message(title, rows)
    last: dict[str, Any] = {"ok": False, "error": "no attempt"}
    for attempt in range(MAX_DIAGNOSE_RETRIES + 1):
        if attempt > 0:
            print(f"  pool/overall retry {attempt}/{MAX_DIAGNOSE_RETRIES} after {RETRY_DELAY_SEC}s …", flush=True)
            time.sleep(RETRY_DELAY_SEC)
        chat = chat_diagnose(agent_base, runtime_key, mcp_token, msg)
        if not chat.get("ok"):
            sid_fail = str(chat.get("session_id") or "").strip()
            if sid_fail:
                rec = pool_jev_from_session(agent_base, runtime_key, mcp_token, sid_fail)
                if rec.get("ok"):
                    rec["attempt"] = attempt + 1
                    return rec
            last = {
                "ok": False,
                "error": chat.get("error") or "chat failed",
                "attempt": attempt + 1,
                "session_id": sid_fail or None,
            }
            print(
                f"  pool/overall attempt {attempt + 1} chat failed: {(last.get('error') or '')[:240]}",
                flush=True,
            )
            continue
        session_id = str(chat.get("session_id") or "").strip()
        run_id = str(chat.get("run_id") or "").strip()
        run_id = ensure_diagnose_run_id(
            agent_base,
            runtime_key,
            mcp_token,
            session_id,
            run_id,
            polls=POOL_ENSURE_RUN_ID_POLLS,
            poll_wait_sec=POOL_ENSURE_RUN_ID_WAIT_SEC,
        )
        jev_reason = str(chat.get("reply") or "").strip()
        res = fetch_diagnose_result(agent_base, runtime_key, mcp_token, session_id, run_id)
        if res.get("ok"):
            data = res["data"]
            if isinstance(data, dict) and isinstance(data.get("data"), dict):
                data = data["data"]
            jr = str(data.get("eval_judgment") or "").strip()
            if len(jr) >= MIN_JEV_REASON_LEN:
                jev_reason = jr
        detail_ref = ""
        if res.get("ok"):
            detail_ref = pick_detail_ref(res["data"] if isinstance(res.get("data"), dict) else {})
        direction = parse_direction_from_compose(jev_reason)
        if session_id and run_id and detail_ref:
            d2, _ = lab_detail_jev(
                agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref
            )
            if d2:
                direction = d2
        if (
            len(jev_reason) < MIN_JEV_REASON_LEN
            or not direction
            or not session_id
            or not run_id
        ):
            err_detail = (
                f"incomplete pool/overall jev"
                f" reason_len={len(jev_reason)} direction={direction!r}"
                f" session={bool(session_id)} run_id={run_id!r}"
            )
            last = {
                "ok": False,
                "error": err_detail,
                "attempt": attempt + 1,
                "session_id": session_id,
                "run_id": run_id,
            }
            print(f"  pool/overall attempt {attempt + 1} incomplete: {err_detail[:280]}", flush=True)
            continue
        return {
            "ok": True,
            "jev_direction": direction,
            "jev_reason": format_pool_jev_reason(jev_reason),
            "session_id": session_id,
            "run_id": run_id,
            "attempt": attempt + 1,
        }
    return last


def parse_direction_from_compose(text: str) -> str:
    t = (text or "").strip()
    if not t:
        return "neutral"
    if re.search(r"(偏空|看空|谨慎空|做空|short)", t, re.I):
        return "short"
    if re.search(r"(偏多|看多|做多|long)", t, re.I):
        return "long"
    return "neutral"


def _filter_grouped_symbols(
    grouped: dict[str, list[dict[str, Any]]], symbols: list[str]
) -> dict[str, list[dict[str, Any]]]:
    want = {s.strip() for s in symbols if s.strip()}
    if not want:
        return grouped
    out: dict[str, list[dict[str, Any]]] = {}
    for pool_name, items in grouped.items():
        kept = [
            it
            for it in items
            if str(it.get("symbol") or "").strip() in want
        ]
        if kept:
            out[pool_name] = kept
    return out


def run_batch(
    workflow: str,
    macro_base: str,
    macro_key: str,
    agent_base: str,
    batch_date: str,
    dry_run: bool,
    *,
    orchestration_date: str | None = None,
    symbol_filter: list[str] | None = None,
) -> tuple[int, dict[str, Any] | None, dict[str, Any] | None]:
    ledger_date = (orchestration_date or batch_date).strip()
    api_workflow = resolve_batch_workflow(workflow)
    runtime_key, mcp_token = load_geegoo_env()
    pool = fetch_monitor_pool(macro_base, macro_key)
    grouped = group_symbols_for_workflow(pool, workflow)
    if symbol_filter:
        grouped = _filter_grouped_symbols(grouped, symbol_filter)
    if not grouped:
        print(f"no symbols for workflow={workflow}", file=sys.stderr)
        return 2, None, None

    expected_pool_syms = monitor_pool_symbols(grouped)

    started = rfc3339_now()
    pool_wf = api_workflow if api_workflow in WORKFLOW_MONITOR_POOL else workflow
    batch_run_id = ""
    jev_writer: MacroJevResultsWriter | None = None
    if not dry_run and pool_wf in WORKFLOW_MONITOR_POOL:
        jev_writer = MacroJevResultsWriter(
            macro_base,
            macro_key,
            batch_date,
            pool_wf,
            "",
            started,
            http_json,
        )
    batch_pools: list[dict[str, Any]] = []
    ingest_items: list[dict[str, Any]] = []
    ok_count = 0
    fail_count = 0
    total = sum(len(v) for v in grouped.values())

    for pool_name, items in grouped.items():
        symbols: list[str] = []
        pool_items: list[dict[str, Any]] = []
        for it in items:
            sym = str(it.get("symbol") or "").strip()
            name = str(it.get("name") or sym).strip()
            symbols.append(sym)
            pool_items.append(
                {
                    "symbol": sym,
                    "name": name,
                    "success": False,
                    "error": DIAGNOSING_ERROR,
                }
            )
        batch_pools.append({"name": pool_name, "symbols": symbols, "items": pool_items})

    pool_jev_out: dict[str, Any] | None = None

    def publish_progress(
        done_label: str,
        *,
        final: bool = False,
        pool_jev: dict[str, Any] | None = None,
    ) -> None:
        nonlocal batch_run_id
        if dry_run:
            return
        in_progress = not final and (ok_count + fail_count < total)
        summary = (
            f"{workflow} {done_label} ok={ok_count} fail={fail_count}/{total}"
            if not in_progress or final
            else f"{workflow} 诊断中 {ok_count + fail_count}/{total} ok={ok_count} fail={fail_count}"
        )
        sym_ok = fail_count == 0 and ok_count == total and total > 0
        body: dict[str, Any] = {
            "workflow": api_workflow,
            "date": batch_date,
            "started_at": started,
            "finished_at": rfc3339_now(),
            "success": False if in_progress else sym_ok,
            "summary": summary,
            "pools": batch_pools,
        }
        if pool_jev:
            body["pool_jev"] = pool_jev
        rid = post_batch_run(macro_base, macro_key, body)
        if rid:
            if not batch_run_id:
                batch_run_id = rid
            if jev_writer:
                jev_writer.bind_batch_run(
                    batch_run_id,
                    batch_started_at=str(body.get("started_at") or started),
                )
            print(f"batch progress id={rid} · {summary}", flush=True)

    if not dry_run:
        publish_progress("started", final=False)
        if jev_writer and not batch_run_id:
            print(
                "jev-results symbol/pool PUT disabled: first batch/runs did not return run id",
                file=sys.stderr,
            )
            jev_writer = None

    sym_index: dict[str, tuple[int, int]] = {}
    for pi, pool in enumerate(batch_pools):
        for ii, sym in enumerate(pool["symbols"]):
            sym_index[sym] = (pi, ii)

    flat: list[tuple[str, dict[str, Any]]] = []
    for pool_name, items in grouped.items():
        for it in items:
            flat.append((pool_name, it))

    for pool_name, it in flat:
        sym = str(it.get("symbol") or "").strip()
        name = str(it.get("name") or sym).strip()
        pi, ii = sym_index[sym]
        print(f"[{workflow}] diagnose {sym} ({name}) …", flush=True)
        batch_pools[pi]["items"][ii] = {
            "symbol": sym,
            "name": name,
            "success": False,
            "error": DIAGNOSING_ERROR,
        }
        publish_progress(sym, final=False)

        if dry_run:
            batch_pools[pi]["items"][ii] = {
                "symbol": sym,
                "name": name,
                "success": True,
                "jev_direction": "neutral",
            }
            ok_count += 1
            continue

        outcome = diagnose_symbol_with_retries(agent_base, runtime_key, mcp_token, name, sym)
        if outcome.get("ok"):
            ok_count += 1
            session_id = str(outcome.get("session_id") or "")
            run_id = str(outcome.get("run_id") or "")
            jev_reason = str(outcome.get("jev_reason") or "")
            detail_ref = str(outcome.get("detail_ref") or "")
            direction = str(outcome.get("direction") or "")
            conf = outcome.get("confidence")
            lab_summary = outcome.get("lab_episode_summary")
            row = {
                "symbol": sym,
                "name": name,
                "success": True,
                "jev_direction": direction,
                "jev_reason": jev_reason or None,
                "detail_ref": detail_ref or None,
                "confidence": conf,
                "agent_session_id": session_id or None,
                "agent_run_id": run_id or None,
            }
            if isinstance(lab_summary, dict) and lab_summary:
                row["lab_episode_summary"] = lab_summary
            batch_pools[pi]["items"][ii] = row
            ingest_item: dict[str, Any] = {
                "code": sym,
                "jev_direction": direction,
                "jev_reason": jev_reason,
                "detail_ref": detail_ref,
                "agent_session_id": session_id,
                "agent_run_id": run_id,
                "lab_supported": True,
            }
            if isinstance(lab_summary, dict) and lab_summary:
                ingest_item["lab_episode_summary"] = lab_summary
            ingest_items.append(ingest_item)
            print(
                f"  OK jev={direction} ref={detail_ref} run={run_id[:24] if run_id else ''}",
                flush=True,
            )
        else:
            fail_count += 1
            err = str(outcome.get("error") or "diagnose failed")
            batch_pools[pi]["items"][ii] = {
                "symbol": sym,
                "name": name,
                "success": False,
                "error": err,
                "agent_session_id": outcome.get("session_id"),
                "agent_run_id": outcome.get("run_id"),
                "detail_ref": outcome.get("detail_ref"),
                "jev_reason": outcome.get("jev_reason"),
            }
            print(f"  FAIL: {err[:240]}", flush=True)

        if jev_writer:
            jev_writer.put_symbol_from_batch_item(
                batch_pools[pi]["items"][ii],
                completed_at=rfc3339_now(),
            )

        publish_progress(sym, final=False)

    summary = f"{workflow} ok={ok_count} fail={fail_count}/{total}; ingest upserted={len(ingest_items)} sent={len(ingest_items)}"
    print(summary, flush=True)

    if dry_run:
        return (0 if fail_count == 0 else 1), None, None

    macro_headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {macro_key}",
    }
    if ingest_items:
        status, data, raw = http_json(
            "POST",
            f"{macro_base.rstrip('/')}/macro/decision/inline/ingest",
            macro_headers,
            {"date": batch_date, "items": ingest_items},
            timeout=120,
        )
        if status != 200:
            print(f"ingest HTTP {status}: {raw[:400]}", file=sys.stderr)
            return 1, None, None
        upserted = (data.get("data") or {}).get("upserted", data.get("upserted"))
        print(f"ingest upserted={upserted}", flush=True)

    batch_run_ok = fail_count == 0 and ok_count == total and total > 0
    pool_synth_rows: list[dict[str, Any]] = []
    pool_complete = False
    pool_skip_reason = ""
    if batch_run_ok and pool_wf in WORKFLOW_MONITOR_POOL:
        pool_synth_rows, pool_complete, pool_skip_reason = resolve_pool_synthesis_rows(
            macro_base,
            macro_key,
            api_workflow,
            batch_date,
            expected_pool_syms,
        )

    defer_pool = workflow in BOND_SEGMENT_WORKFLOWS
    pool_final_published = False
    if (
        not dry_run
        and not defer_pool
        and batch_run_ok
        and pool_complete
        and pool_wf in WORKFLOW_MONITOR_POOL
    ):
        label = WORKFLOW_MONITOR_POOL[pool_wf]
        print(
            f"[{workflow}] pool JEV synthesize ({label}, {len(pool_synth_rows)} symbols) …",
            flush=True,
        )
        try:
            syn = synthesize_jev_with_retries(
                agent_base, runtime_key, mcp_token, label, pool_synth_rows
            )
            finished = rfc3339_now()
            valid, val_err = synthesis_pool_jev_valid(syn, min_reason_len=MIN_JEV_REASON_LEN)
            if valid:
                pool_jev_out = {
                    "success": True,
                    "jev_direction": syn.get("jev_direction"),
                    "jev_reason": syn.get("jev_reason"),
                    "agent_session_id": syn.get("session_id"),
                    "agent_run_id": syn.get("run_id"),
                    "finished_at": finished,
                    "attempt": syn.get("attempt"),
                }
                print(f"  pool JEV ok={pool_jev_out.get('jev_direction')}", flush=True)
            elif syn.get("ok"):
                err = val_err or "incomplete pool synthesis fields"
                pool_jev_out = {
                    "success": False,
                    "error": err,
                    "finished_at": finished,
                    "agent_session_id": syn.get("session_id"),
                    "agent_run_id": syn.get("run_id"),
                }
                print(f"  pool JEV incomplete: {err[:240]}", flush=True)
            else:
                err = str(syn.get("error") or "pool jev failed")
                pool_jev_out = {
                    "success": False,
                    "error": err,
                    "attempt": syn.get("attempt"),
                    "finished_at": finished,
                    "agent_session_id": syn.get("session_id"),
                }
                print(f"  pool JEV FAIL: {err[:240]}", flush=True)
        except Exception as exc:
            finished = rfc3339_now()
            pool_jev_out = {
                "success": False,
                "error": f"{type(exc).__name__}: {exc}",
                "finished_at": finished,
            }
            traceback.print_exc()
            print(f"  pool JEV EXCEPTION: {exc}", flush=True)
        finally:
            if not dry_run:
                publish_progress("finished", final=True, pool_jev=pool_jev_out)
                pool_final_published = True
    elif (
        not dry_run
        and batch_run_ok
        and pool_wf in WORKFLOW_MONITOR_POOL
        and not pool_complete
        and pool_skip_reason
    ):
        print(
            f"  pool JEV skipped: {pool_skip_reason}; "
            "use --pool-jev-only when the full monitor pool is success",
            flush=True,
        )
    if not dry_run and not pool_final_published:
        publish_progress("finished", final=True, pool_jev=pool_jev_out)
    if jev_writer and not defer_pool:
        all_items = [it for p in batch_pools for it in (p.get("items") or [])]
        sym_all_ok = pool_complete
        pool_items_for_put = pool_synth_rows if pool_complete else all_items
        extra_canon: dict[str, str] | None = None
        if pool_complete:
            extra_canon, c_err = fetch_canonical_symbol_result_ids(
                macro_base, macro_key, http_json, batch_date, api_workflow
            )
            if c_err:
                print(f"  canonical symbols: {c_err}", flush=True)
        if batch_run_id:
            jev_writer.bind_batch_run(batch_run_id)
        jev_writer.put_pool_from_synthesis(
            pool_jev_out,
            symbol_items=pool_items_for_put,
            all_symbols_ok=sym_all_ok,
            completed_at=rfc3339_now(),
            expected_symbols=expected_pool_syms,
            extra_symbol_result_ids=extra_canon,
        )
        if jev_writer.pool_result_id and jev_writer.batch_identity_ready():
            pool_st = "success" if pool_jev_out and pool_jev_out.get("success") else "failed"
            record_pool_ledger(
                ledger_date,
                pool_wf,
                result_id=jev_writer.pool_result_id,
                source_batch_id=jev_writer.source_batch_id,
                batch_started_at=jev_writer.batch_started_at,
                status=pool_st,
                cron_workflow=workflow,
            )
    pool_input = None
    if pool_jev_out and pool_jev_out.get("success"):
        pool_input = {
            "label": WORKFLOW_MONITOR_POOL.get(pool_wf, pool_wf),
            "jev_direction": pool_jev_out.get("jev_direction"),
            "jev_reason": pool_jev_out.get("jev_reason"),
        }
    rc = 0 if fail_count == 0 and (pool_jev_out is None or pool_jev_out.get("success")) else 1
    if jev_writer and jev_writer.has_errors():
        rc = max(rc, 1)
    if (
        not dry_run
        and fail_count == 0
        and ok_count == total
        and workflow in BOND_SEGMENT_WORKFLOWS
    ):
        rc = max(rc, try_finalize_bond_pool_jev(macro_base, macro_key, agent_base, ledger_date))
    pool_meta = None
    if jev_writer and jev_writer.pool_result_id:
        pool_meta = {
            "pool_workflow": pool_wf,
            "result_id": jev_writer.pool_result_id,
            "source_batch_id": jev_writer.source_batch_id,
        }
    return rc, pool_input, pool_meta


def run_pool_jev_only(
    workflow: str,
    macro_base: str,
    macro_key: str,
    agent_base: str,
    batch_date: str,
    *,
    recover_session_id: str = "",
) -> int:
    """Re-run or recover pool-level JEV without re-diagnosing symbols (e.g. after SSE drop)."""
    api_workflow = resolve_batch_workflow(workflow)
    if api_workflow not in WORKFLOW_MONITOR_POOL:
        print(f"workflow {workflow} has no pool JEV step", file=sys.stderr)
        return 2
    runtime_key, mcp_token = load_geegoo_env()
    rows = merged_batch_symbol_rows(macro_base, macro_key, api_workflow, batch_date)
    if not rows:
        print(
            f"no successful symbol rows on {batch_date} for workflow={api_workflow}",
            file=sys.stderr,
        )
        return 2
    label = WORKFLOW_MONITOR_POOL[api_workflow]
    started = rfc3339_now()
    pool_name = label
    batch_pools = [
        {
            "name": pool_name,
            "symbols": [str(r.get("symbol") or "") for r in rows],
            "items": rows,
        }
    ]
    if recover_session_id.strip():
        print(f"[{workflow}] recover pool JEV from session {recover_session_id} …", flush=True)
        syn = pool_jev_from_session(
            agent_base, runtime_key, mcp_token, recover_session_id.strip()
        )
    else:
        print(f"[{workflow}] pool JEV only ({label}) …", flush=True)
        syn = synthesize_jev_with_retries(
            agent_base, runtime_key, mcp_token, label, rows
        )
    finished = rfc3339_now()
    if syn.get("ok"):
        pool_jev = {
            "success": True,
            "jev_direction": syn.get("jev_direction"),
            "jev_reason": syn.get("jev_reason"),
            "agent_session_id": syn.get("session_id"),
            "agent_run_id": syn.get("run_id"),
            "finished_at": finished,
            "attempt": syn.get("attempt"),
            "recovered": syn.get("recovered"),
        }
        sym_ok = all(r.get("success") for r in rows)
        body = {
            "workflow": api_workflow,
            "date": batch_date,
            "started_at": started,
            "finished_at": finished,
            "success": sym_ok and True,
            "summary": f"{api_workflow} pool JEV finalize ok={pool_jev.get('jev_direction')}",
            "pools": batch_pools,
            "pool_jev": pool_jev,
        }
        pool_run_id = post_batch_run(macro_base, macro_key, body)
        writer = MacroJevResultsWriter(
            macro_base,
            macro_key,
            batch_date,
            api_workflow,
            pool_run_id or "",
            started,
            http_json,
        )
        if pool_run_id:
            writer.bind_batch_run(pool_run_id)
        canon, c_err = fetch_canonical_symbol_result_ids(
            macro_base, macro_key, http_json, batch_date, api_workflow
        )
        if c_err:
            print(f"  canonical symbols: {c_err}", flush=True)
        writer.put_pool_from_synthesis(
            pool_jev,
            symbol_items=rows,
            all_symbols_ok=sym_ok,
            completed_at=finished,
            extra_symbol_result_ids=canon,
            expected_symbols=[str(r.get("symbol") or "") for r in rows],
        )
        if writer.pool_result_id and writer.batch_identity_ready():
            record_pool_ledger(
                batch_date,
                api_workflow,
                result_id=writer.pool_result_id,
                source_batch_id=writer.source_batch_id,
                batch_started_at=writer.batch_started_at,
                status="success" if sym_ok else "failed",
                cron_workflow=f"{workflow}:pool-jev-only",
            )
        print(f"  pool JEV ok={pool_jev.get('jev_direction')}", flush=True)
        rc = 0
        if writer.has_errors():
            rc = 1
        return rc
    err = str(syn.get("error") or "pool jev failed")
    pool_jev = {
        "success": False,
        "error": err,
        "finished_at": finished,
        "agent_session_id": syn.get("session_id"),
    }
    body = {
        "workflow": api_workflow,
        "date": batch_date,
        "started_at": started,
        "finished_at": finished,
        "success": False,
        "summary": f"{api_workflow} pool JEV finalize failed",
        "pools": batch_pools,
        "pool_jev": pool_jev,
    }
    pool_run_id = post_batch_run(macro_base, macro_key, body)
    writer = MacroJevResultsWriter(
        macro_base,
        macro_key,
        batch_date,
        api_workflow,
        pool_run_id or "",
        started,
        http_json,
    )
    if pool_run_id:
        writer.bind_batch_run(pool_run_id)
    canon, c_err = fetch_canonical_symbol_result_ids(
        macro_base, macro_key, http_json, batch_date, api_workflow
    )
    if c_err:
        print(f"  canonical symbols: {c_err}", flush=True)
    writer.put_pool_from_synthesis(
        pool_jev,
        symbol_items=rows,
        all_symbols_ok=False,
        completed_at=finished,
        extra_symbol_result_ids=canon,
    )
    print(f"  pool JEV FAIL: {err[:240]}", flush=True)
    rc = 1
    if writer.has_errors():
        rc = 1
    return rc


def refresh_ledger_before_overall(
    macro_base: str,
    macro_key: str,
    agent_base: str,
    orchestration_date: str,
    dry_run: bool,
) -> dict[str, dict[str, str]]:
    """At 17:00 overall: rerun cron workflows for pools not yet successful in the ledger."""
    ledger = load_pool_ledger(orchestration_date)
    missing = pools_needing_rerun(ledger)
    if not missing:
        return ledger
    printed = False
    seen: set[str] = set()
    for pool in missing:
        crons = POOL_RERUN_CRON_WORKFLOWS.get(pool, (pool,))
        for cron in crons:
            if cron in seen:
                continue
            seen.add(cron)
            macro_d = macro_batch_date_for_cron(cron, orchestration_date)
            if not printed:
                print(
                    f"[overall] ledger incomplete orch={orchestration_date}; "
                    f"rerun pools before aggregation …",
                    flush=True,
                )
                printed = True
            print(
                f"[overall] rerun cron={cron} macro_date={macro_d} (pool={pool}) …",
                flush=True,
            )
            run_batch(
                cron,
                macro_base,
                macro_key,
                agent_base,
                macro_d,
                dry_run,
                orchestration_date=orchestration_date,
            )
        if pool == "bond":
            try_finalize_bond_pool_jev(
                macro_base, macro_key, agent_base, orchestration_date
            )
    return load_pool_ledger(orchestration_date)


def run_overall_batch(
    macro_base: str,
    macro_key: str,
    agent_base: str,
    batch_date: str,
    pool_inputs: list[dict[str, Any]],
    dry_run: bool,
    *,
    pool_ledger: dict[str, dict[str, str]] | None = None,
    skip_pool_rerun: bool = False,
) -> int:
    if dry_run:
        return 0
    ledger = pool_ledger
    if ledger is None and not skip_pool_rerun:
        ledger = refresh_ledger_before_overall(
            macro_base, macro_key, agent_base, batch_date, dry_run
        )
    if ledger is None:
        ledger = load_pool_ledger(batch_date)
    pool_refs, ref_err = overall_inputs_from_ledger(
        macro_base, macro_key, http_json, batch_date, ledger
    )
    synth_rows, syn_err = pool_synth_rows_from_ledger(
        macro_base, macro_key, http_json, batch_date, ledger
    )

    def write_overall_blocked(reason: str) -> int:
        started = rfc3339_now()
        finished = rfc3339_now()
        overall_run_id = post_batch_run(
            macro_base,
            macro_key,
            {
                "workflow": "overall",
                "date": batch_date,
                "started_at": started,
                "finished_at": finished,
                "success": False,
                "summary": f"overall blocked: {reason[:120]}",
                "pools": [{"name": "宏观整体", "symbols": [], "items": []}],
                "pool_jev": {"success": False, "error": reason},
            },
        )
        if not overall_run_id:
            print("overall blocked: batch/runs id missing", file=sys.stderr)
            return 2
        out = put_overall_result(
            macro_base,
            macro_key,
            http_json,
            batch_date=batch_date,
            source_batch_id=overall_run_id,
            batch_started_at=started,
            completed_at=finished,
            pool_jev={"success": False, "error": reason},
            pool_inputs=pool_refs or [],
        )
        if not out.ok:
            print(
                f"overall jev-results PUT not accepted: {out.reason or out.error}",
                flush=True,
            )
        return 2

    if not pool_refs or ref_err:
        return write_overall_blocked(ref_err or "orchestration ledger incomplete")
    if not synth_rows or syn_err:
        return write_overall_blocked(syn_err or "ledger pool rows not synthesizable")

    runtime_key, mcp_token = load_geegoo_env()
    started = rfc3339_now()
    print("[overall] macro JEV synthesize (ledger-bound pool rows) …", flush=True)
    syn = synthesize_jev_with_retries(
        agent_base, runtime_key, mcp_token, "宏观整体", synth_rows
    )
    finished = rfc3339_now()
    pool_jev: dict[str, Any]
    valid, val_err = synthesis_pool_jev_valid(syn, min_reason_len=MIN_JEV_REASON_LEN)
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
            "error": val_err or str(syn.get("error") or "overall jev failed"),
            "attempt": syn.get("attempt"),
            "finished_at": finished,
            "agent_session_id": syn.get("session_id"),
        }
    body = {
        "workflow": "overall",
        "date": batch_date,
        "started_at": started,
        "finished_at": finished,
        "success": bool(pool_jev.get("success")),
        "summary": "overall JEV",
        "pools": [{"name": "宏观整体", "symbols": [], "items": []}],
        "pool_jev": pool_jev,
    }
    overall_run_id = post_batch_run(macro_base, macro_key, body)
    ok_overall = bool(pool_jev.get("success")) and len(pool_refs) == len(
        JEV_PIPELINE_WORKFLOWS
    )
    if not ok_overall:
        pool_jev = {
            **pool_jev,
            "success": False,
            "error": pool_jev.get("error") or "overall synthesis incomplete",
        }
    if not overall_run_id:
        print("overall: batch/runs id missing; canonical PUT blocked", file=sys.stderr)
        return 1
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
    if not out.ok:
        print(
            f"overall jev-results PUT not accepted: {out.reason or out.error}",
            flush=True,
        )
    return 0 if ok_overall and out.ok else 1


def main() -> int:
    wf_choices = (
        list(JEV_PIPELINE_WORKFLOWS)
        + ["overall", "all", *BOND_SEGMENT_WORKFLOWS]
    )
    p = argparse.ArgumentParser(description="Run macro monitor pool Lab batch workflow")
    p.add_argument("--workflow", choices=wf_choices, required=True)
    p.add_argument(
        "--date",
        default="auto",
        help="YYYY-MM-DD Shanghai orchestration window, or auto (default)",
    )
    p.add_argument("--macro-base", default=os.environ.get("MACRO_API_BASE", DEFAULT_MACRO))
    p.add_argument(
        "--macro-key",
        default=os.environ.get("MACRO_API_KEY", os.environ.get("GEEGOO_SIGNAL_MACRO_API_KEY", "")),
    )
    p.add_argument(
        "--agent-base",
        default=os.environ.get("GEEGOO_AGENT_OP_HTTP_URL", DEFAULT_AGENT),
    )
    p.add_argument("--dry-run", action="store_true")
    p.add_argument(
        "--pool-jev-only",
        action="store_true",
        help="Skip symbol diagnose; synthesize or recover pool JEV from existing batch rows",
    )
    p.add_argument(
        "--recover-session-id",
        default="",
        help="With --pool-jev-only: load pool JEV from an existing agent chat session",
    )
    p.add_argument(
        "--symbols",
        default="",
        help="Comma-separated monitor symbols only (e.g. 000001.SH for 上证)",
    )
    p.add_argument(
        "--rerun-missing-pools",
        action="store_true",
        help="With --workflow overall: rerun ledger-missing pools before synthesis (default: ledger-only, no auto rerun)",
    )
    args = p.parse_args()
    if not args.macro_key.strip():
        print("MACRO_API_KEY required", file=sys.stderr)
        return 2

    symbol_filter: list[str] | None = None
    if str(args.symbols or "").strip():
        symbol_filter = [
            s.strip() for s in str(args.symbols).split(",") if s.strip()
        ]

    macro_key = args.macro_key.strip()
    orchestration_date, macro_batch_date = resolve_run_dates(args.workflow, args.date)
    pool_inputs: list[dict[str, Any]] = []
    ledger = load_pool_ledger(orchestration_date)
    rc = 0
    if args.workflow == "all":
        for wf in JEV_PIPELINE_WORKFLOWS:
            macro_d = macro_batch_date_for_cron(wf, orchestration_date)
            code, pool_in, _pool_meta = run_batch(
                wf,
                args.macro_base,
                macro_key,
                args.agent_base,
                macro_d,
                args.dry_run,
                orchestration_date=orchestration_date,
            )
            rc = max(rc, code)
            if pool_in:
                pool_inputs.append(pool_in)
        ledger = load_pool_ledger(orchestration_date)
        code = run_overall_batch(
            args.macro_base,
            macro_key,
            args.agent_base,
            orchestration_date,
            pool_inputs,
            args.dry_run,
            pool_ledger=ledger,
            skip_pool_rerun=True,
        )
        rc = max(rc, code)
        return rc
    if args.workflow == "overall":
        return run_overall_batch(
            args.macro_base,
            macro_key,
            args.agent_base,
            orchestration_date,
            pool_inputs,
            args.dry_run,
            pool_ledger=None,
            skip_pool_rerun=not args.rerun_missing_pools,
        )
    if args.pool_jev_only:
        if args.workflow in ("all", "overall") or args.workflow in BOND_SEGMENT_WORKFLOWS:
            print("--pool-jev-only requires a single monitor pool workflow", file=sys.stderr)
            return 2
        return run_pool_jev_only(
            args.workflow,
            args.macro_base,
            macro_key,
            args.agent_base,
            macro_batch_date,
            recover_session_id=args.recover_session_id,
        )
    code, _, _ = run_batch(
        args.workflow,
        args.macro_base,
        macro_key,
        args.agent_base,
        macro_batch_date,
        args.dry_run,
        orchestration_date=orchestration_date,
        symbol_filter=symbol_filter,
    )
    return code


if __name__ == "__main__":
    raise SystemExit(main())
