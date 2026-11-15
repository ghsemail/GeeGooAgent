#!/usr/bin/env python3
"""宏观监控池日批：逐标的 signal_diagnose → macro inline/ingest + batch/runs."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from collections import defaultdict
from datetime import datetime, timedelta, timezone
from typing import Any

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
LEGACY_WORKFLOWS = ("asia", "us", "crypto")

CHAT_TIMEOUT = int(os.environ.get("MACRO_BATCH_CHAT_TIMEOUT", "360"))
MIN_JEV_REASON_LEN = 20
MAX_DIAGNOSE_RETRIES = 3
RETRY_DELAY_SEC = int(os.environ.get("MACRO_BATCH_RETRY_DELAY_SEC", "600"))
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


def http_json(
    method: str,
    url: str,
    headers: dict[str, str],
    body: dict[str, Any] | None = None,
    timeout: int = 120,
) -> tuple[int, Any, str]:
    data = json.dumps(body or {}, ensure_ascii=False).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
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


def shanghai_today() -> str:
    tz = timezone(timedelta(hours=8))
    return datetime.now(tz).strftime("%Y-%m-%d")


def rfc3339_now() -> str:
    tz = timezone(timedelta(hours=8))
    return datetime.now(tz).isoformat(timespec="seconds")


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
    pool_name = WORKFLOW_MONITOR_POOL.get(workflow)
    if workflow in LEGACY_WORKFLOWS:
        return _group_symbols_legacy(pool, workflow)
    if not pool_name:
        return {}
    items: list[dict[str, Any]] = []
    for it in pool:
        name = (it.get("monitor_reason") or "").strip()
        if name != pool_name:
            continue
        sym = (it.get("symbol") or "").strip()
        if sym:
            items.append(it)
    if not items:
        return {}
    return {pool_name: items}


def _group_symbols_legacy(pool: list[dict[str, Any]], workflow: str) -> dict[str, list[dict[str, Any]]]:
    legacy_pools: dict[str, set[str]] = {
        "asia": {"A股池", "债券利率池", "港股池", "波动率池"},
        "us": {"美股池", "商品货币池", "债券利率池", "波动率池"},
        "crypto": {"虚拟货币池"},
    }
    allowed = legacy_pools.get(workflow, set())
    sym_filter = lambda pn, sym: True  # noqa: E731
    pools: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for it in pool:
        name = (it.get("monitor_reason") or "").strip()
        if name not in allowed:
            continue
        sym = (it.get("symbol") or "").strip().upper()
        if not sym:
            continue
        if name == "债券利率池":
            if workflow == "asia" and not sym.endswith(".SH"):
                continue
            if workflow == "us" and not sym.endswith(".US"):
                continue
        if name == "波动率池":
            if workflow == "asia" and "HSIL" not in sym:
                continue
            if workflow == "us" and not sym.startswith("VIX"):
                continue
        pools[name].append(it)
    return dict(pools)


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
) -> dict[str, Any]:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-Approve-Writes": "true",
        "X-MCP-Token": mcp_token,
    }
    body = {
        "message": message,
        "mcp_token": mcp_token,
        "workflow_options": WORKFLOW_OPTIONS,
    }
    url = f"{agent_base.rstrip('/')}/v1/chat/stream"
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    sid = ""
    run_id = ""
    turn_end: dict[str, Any] = {}
    event = ""
    try:
        with urllib.request.urlopen(req, timeout=CHAT_TIMEOUT) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        return {"ok": False, "error": f"chat/stream HTTP {e.code}: {raw[:500]}"}
    for line in raw.splitlines():
        if line.startswith("event:"):
            event = line[6:].strip()
        elif line.startswith("data:"):
            payload = line[5:].strip()
            if event == "connected":
                try:
                    sid = json.loads(payload).get("session_id", sid) or sid
                except json.JSONDecodeError:
                    pass
            elif event == "signal_diagnose_result_ready":
                try:
                    ready = json.loads(payload)
                    run_id = str(ready.get("run_id") or run_id) or run_id
                except json.JSONDecodeError:
                    pass
            elif event == "turn_end":
                try:
                    turn_end = json.loads(payload)
                except json.JSONDecodeError:
                    turn_end = {"raw": payload}
    failed = bool(turn_end.get("failed"))
    err = str(turn_end.get("error") or "")
    if failed or err:
        return {"ok": False, "session_id": sid, "error": err or "turn failed"}
    reply = str(turn_end.get("assistant_text") or "").strip()
    return {"ok": True, "session_id": sid, "run_id": run_id, "reply": reply}


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
    # Result API often inlines card summaries: lab is { "buy:9": {...}, "sell:43": {...} }.
    if isinstance(lab, dict):
        inline = _best_episode_ref([str(k) for k in lab.keys()])
        if inline:
            return inline
    summaries = lab.get("lab_episode_summaries")
    if isinstance(summaries, dict):
        picked = _best_episode_ref([str(k) for k in summaries])
        if picked:
            return picked
    signal_eval = result.get("signal_eval") or {}
    eps: list[str] = []
    for ep in signal_eval.get("buy_details") or []:
        if isinstance(ep, dict):
            eps.append(str(ep.get("detail_ref") or ep.get("ref") or ""))
    for ep in signal_eval.get("sell_details") or []:
        if isinstance(ep, dict):
            eps.append(str(ep.get("detail_ref") or ep.get("ref") or ""))
    return _best_episode_ref(eps)


def is_valid_detail_ref(ref: str) -> bool:
    r = (ref or "").strip()
    return r.startswith("buy:") or r.startswith("sell:")


def item_jev_complete(
    jev_reason: str,
    detail_ref: str,
    session_id: str,
    run_id: str,
    direction: str,
) -> bool:
    if len((jev_reason or "").strip()) < MIN_JEV_REASON_LEN:
        return False
    if not is_valid_detail_ref(detail_ref):
        return False
    if not (session_id or "").strip() or not (run_id or "").strip():
        return False
    if not (direction or "").strip():
        return False
    return True


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
    direction, conf = ("", None)
    if session_id and run_id and detail_ref:
        direction, conf = lab_detail_jev(
            agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref
        )
    if not direction:
        direction = parse_direction_from_compose(jev_reason)
    if conf is None:
        conf = 1.0 if direction != "neutral" else 0.5

    if not item_jev_complete(jev_reason, detail_ref, session_id, run_id, direction):
        parts: list[str] = []
        if len(jev_reason) < MIN_JEV_REASON_LEN:
            parts.append(f"jev_reason<{MIN_JEV_REASON_LEN}")
        if not is_valid_detail_ref(detail_ref):
            parts.append("missing buy:/sell: detail_ref")
        if not session_id or not run_id:
            parts.append("missing session/run")
        if not direction:
            parts.append("missing direction")
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


def lab_detail_jev(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    run_id: str,
    detail_ref: str,
) -> tuple[str, float | None]:
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
    status, data, _ = http_json(
        "POST",
        f"{agent_base.rstrip('/')}/v1/signal-diagnose/lab-detail",
        headers,
        body,
        timeout=180,
    )
    if status != 200:
        return "", None
    detail = data.get("detail") if isinstance(data.get("detail"), dict) else {}
    unified = detail.get("jev_unified") or data.get("jev_unified") or {}
    if isinstance(unified, dict):
        choice = str(
            unified.get("choice")
            or unified.get("direction")
            or unified.get("recommended_choice")
            or ""
        ).strip().lower()
        conf = unified.get("confidence")
        try:
            conf_f = float(conf) if conf is not None else None
        except (TypeError, ValueError):
            conf_f = None
        direction = ""
        if choice in ("long", "short", "neutral"):
            direction = choice
        elif choice in ("bull", "bear"):
            direction = "long" if choice == "bull" else "short"
        elif "多" in choice or "涨" in choice or "bull" in choice:
            direction = "long"
        elif "空" in choice or "跌" in choice or "bear" in choice:
            direction = "short"
        if direction:
            return direction, conf_f
    return "", None


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
        f"禁止简单计数或涨跌幅规则。\n\n{joined}"
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
            last = {"ok": False, "error": chat.get("error") or "chat failed", "attempt": attempt + 1}
            continue
        session_id = str(chat.get("session_id") or "").strip()
        run_id = str(chat.get("run_id") or "").strip()
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
            d2, _ = lab_detail_jev(agent_base, runtime_key, mcp_token, session_id, run_id, detail_ref)
            if d2:
                direction = d2
        if len(jev_reason) < MIN_JEV_REASON_LEN or not direction:
            last = {
                "ok": False,
                "error": "incomplete pool/overall jev",
                "attempt": attempt + 1,
                "session_id": session_id,
                "run_id": run_id,
            }
            continue
        return {
            "ok": True,
            "jev_direction": direction,
            "jev_reason": jev_reason,
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


def run_batch(
    workflow: str,
    macro_base: str,
    macro_key: str,
    agent_base: str,
    batch_date: str,
    dry_run: bool,
) -> tuple[int, dict[str, Any] | None]:
    runtime_key, mcp_token = load_geegoo_env()
    pool = fetch_monitor_pool(macro_base, macro_key)
    grouped = group_symbols_for_workflow(pool, workflow)
    if not grouped:
        print(f"no symbols for workflow={workflow}", file=sys.stderr)
        return 2, None

    started = rfc3339_now()
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
            "workflow": workflow,
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
            print(f"batch progress id={rid} · {summary}", flush=True)

    if not dry_run:
        publish_progress("started", final=False)

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
            batch_pools[pi]["items"][ii] = row
            ingest_items.append(
                {
                    "code": sym,
                    "jev_direction": direction,
                    "jev_reason": jev_reason,
                    "detail_ref": detail_ref,
                    "agent_session_id": session_id,
                    "agent_run_id": run_id,
                    "lab_supported": True,
                }
            )
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

        publish_progress(sym, final=False)

    summary = f"{workflow} ok={ok_count} fail={fail_count}/{total}; ingest upserted={len(ingest_items)} sent={len(ingest_items)}"
    print(summary, flush=True)

    if dry_run:
        return (0 if fail_count == 0 else 1), None

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
            return 1, None
        upserted = (data.get("data") or {}).get("upserted", data.get("upserted"))
        print(f"ingest upserted={upserted}", flush=True)

    symbol_rows_for_pool: list[dict[str, Any]] = []
    if fail_count == 0 and ok_count == total and total > 0:
        for p in batch_pools:
            for it in p.get("items") or []:
                if it.get("success"):
                    symbol_rows_for_pool.append(it)

    if (
        not dry_run
        and fail_count == 0
        and ok_count == total
        and total > 0
        and workflow in WORKFLOW_MONITOR_POOL
    ):
        label = WORKFLOW_MONITOR_POOL[workflow]
        print(f"[{workflow}] pool JEV synthesize ({label}) …", flush=True)
        syn = synthesize_jev_with_retries(
            agent_base, runtime_key, mcp_token, label, symbol_rows_for_pool
        )
        finished = rfc3339_now()
        if syn.get("ok"):
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
        else:
            err = str(syn.get("error") or "pool jev failed")
            pool_jev_out = {
                "success": False,
                "error": err,
                "attempt": syn.get("attempt"),
                "finished_at": finished,
            }
            print(f"  pool JEV FAIL: {err[:240]}", flush=True)

    if not dry_run:
        publish_progress("finished", final=True, pool_jev=pool_jev_out)
    pool_input = None
    if pool_jev_out and pool_jev_out.get("success"):
        pool_input = {
            "label": WORKFLOW_MONITOR_POOL.get(workflow, workflow),
            "jev_direction": pool_jev_out.get("jev_direction"),
            "jev_reason": pool_jev_out.get("jev_reason"),
        }
    rc = 0 if fail_count == 0 and (pool_jev_out is None or pool_jev_out.get("success")) else 1
    return rc, pool_input


def run_overall_batch(
    macro_base: str,
    macro_key: str,
    agent_base: str,
    batch_date: str,
    pool_inputs: list[dict[str, Any]],
    dry_run: bool,
) -> int:
    if len(pool_inputs) < len(JEV_PIPELINE_WORKFLOWS):
        print(
            f"overall skipped: only {len(pool_inputs)}/{len(JEV_PIPELINE_WORKFLOWS)} pool jev",
            file=sys.stderr,
        )
        return 2
    runtime_key, mcp_token = load_geegoo_env()
    started = rfc3339_now()
    if dry_run:
        return 0
    print("[overall] macro JEV synthesize …", flush=True)
    syn = synthesize_jev_with_retries(
        agent_base, runtime_key, mcp_token, "宏观整体", pool_inputs
    )
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
    else:
        pool_jev = {
            "success": False,
            "error": str(syn.get("error") or "overall jev failed"),
            "attempt": syn.get("attempt"),
            "finished_at": finished,
        }
    body = {
        "workflow": "overall",
        "date": batch_date,
        "started_at": started,
        "finished_at": finished,
        "success": bool(pool_jev.get("success")),
        "summary": "overall JEV",
        "pools": [
            {
                "name": "宏观整体",
                "symbols": [],
                "items": [],
            }
        ],
        "pool_jev": pool_jev,
    }
    post_batch_run(macro_base, macro_key, body)
    return 0 if pool_jev.get("success") else 1


def main() -> int:
    wf_choices = JEV_PIPELINE_WORKFLOWS + ["overall", "all", *LEGACY_WORKFLOWS]
    p = argparse.ArgumentParser(description="Run macro monitor pool Lab batch workflow")
    p.add_argument("--workflow", choices=wf_choices, required=True)
    p.add_argument("--date", default=shanghai_today(), help="YYYY-MM-DD Shanghai")
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
    args = p.parse_args()
    if not args.macro_key.strip():
        print("MACRO_API_KEY required", file=sys.stderr)
        return 2

    macro_key = args.macro_key.strip()
    pool_inputs: list[dict[str, Any]] = []
    rc = 0
    if args.workflow == "all":
        for wf in JEV_PIPELINE_WORKFLOWS:
            code, pool_in = run_batch(
                wf, args.macro_base, macro_key, args.agent_base, args.date, args.dry_run
            )
            rc = max(rc, code)
            if pool_in:
                pool_inputs.append(pool_in)
        code = run_overall_batch(
            args.macro_base, macro_key, args.agent_base, args.date, pool_inputs, args.dry_run
        )
        rc = max(rc, code)
        return rc
    if args.workflow == "overall":
        return run_overall_batch(
            args.macro_base, macro_key, args.agent_base, args.date, pool_inputs, args.dry_run
        )
    code, _ = run_batch(
        args.workflow, args.macro_base, macro_key, args.agent_base, args.date, args.dry_run
    )
    return code


if __name__ == "__main__":
    raise SystemExit(main())
