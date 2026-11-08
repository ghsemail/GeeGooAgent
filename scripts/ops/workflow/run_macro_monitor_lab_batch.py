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

WORKFLOW_POOLS: dict[str, set[str]] = {
    "asia": {"A股池", "债券利率池", "港股池", "波动率池"},
    "us": {"美股池", "商品货币池", "债券利率池", "波动率池"},
    "crypto": {"虚拟货币池"},
}

CHAT_TIMEOUT = int(os.environ.get("MACRO_BATCH_CHAT_TIMEOUT", "360"))
MIN_JEV_REASON_LEN = 20
MAX_DIAGNOSE_RETRIES = 3
RETRY_DELAY_SEC = 30
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


def workflow_includes_symbol(workflow: str, pool_name: str, symbol: str) -> bool:
    sym = symbol.upper()
    if pool_name == "债券利率池":
        if workflow == "asia":
            return sym.endswith(".SH")
        if workflow == "us":
            return sym.endswith(".US")
        return False
    if pool_name == "波动率池":
        if workflow == "asia":
            return "HSIL" in sym
        if workflow == "us":
            return sym.startswith("VIX")
        return False
    return True


def group_symbols_for_workflow(pool: list[dict[str, Any]], workflow: str) -> dict[str, list[dict[str, Any]]]:
    allowed = WORKFLOW_POOLS[workflow]
    pools: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for it in pool:
        name = (it.get("monitor_reason") or "").strip()
        if name not in allowed:
            continue
        sym = (it.get("symbol") or "").strip()
        if not sym or not workflow_includes_symbol(workflow, name, sym):
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


def pick_detail_ref(result: dict[str, Any]) -> str:
    lab = result.get("lab") or {}
    ref = (lab.get("lab_default_detail_ref") or "").strip()
    if ref.startswith(("buy:", "sell:")):
        return ref
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
    status, data, raw = http_json(
        "POST",
        f"{macro_base.rstrip('/')}/macro/batch/runs",
        macro_headers,
        body,
        timeout=120,
    )
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
    unified = data.get("jev_unified") or {}
    if isinstance(unified, dict):
        choice = str(unified.get("choice") or unified.get("direction") or "").strip().lower()
        conf = unified.get("confidence")
        try:
            conf_f = float(conf) if conf is not None else None
        except (TypeError, ValueError):
            conf_f = None
        direction = ""
        if choice in ("long", "short", "neutral"):
            direction = choice
        elif "多" in choice or "涨" in choice:
            direction = "long"
        elif "空" in choice or "跌" in choice:
            direction = "short"
        if direction:
            return direction, conf_f
    return "", None


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
) -> int:
    runtime_key, mcp_token = load_geegoo_env()
    pool = fetch_monitor_pool(macro_base, macro_key)
    grouped = group_symbols_for_workflow(pool, workflow)
    if not grouped:
        print(f"no symbols for workflow={workflow}", file=sys.stderr)
        return 2

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

    def publish_progress(done_label: str, *, final: bool = False) -> None:
        if dry_run:
            return
        in_progress = not final and (ok_count + fail_count < total)
        summary = (
            f"{workflow} {done_label} ok={ok_count} fail={fail_count}/{total}"
            if not in_progress or final
            else f"{workflow} 诊断中 {ok_count + fail_count}/{total} ok={ok_count} fail={fail_count}"
        )
        body: dict[str, Any] = {
            "workflow": workflow,
            "date": batch_date,
            "started_at": started,
            "finished_at": rfc3339_now(),
            "success": False if in_progress else (fail_count == 0 and ok_count > 0),
            "summary": summary,
            "pools": batch_pools,
        }
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
        return 0 if fail_count == 0 else 1

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
            return 1
        upserted = (data.get("data") or {}).get("upserted", data.get("upserted"))
        print(f"ingest upserted={upserted}", flush=True)

    if not dry_run:
        publish_progress("finished", final=True)
    return 0 if fail_count == 0 else 1


def main() -> int:
    p = argparse.ArgumentParser(description="Run macro monitor pool Lab batch workflow")
    p.add_argument("--workflow", choices=["asia", "us", "crypto", "all"], required=True)
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

    workflows = ["asia", "us", "crypto"] if args.workflow == "all" else [args.workflow]
    rc = 0
    for wf in workflows:
        code = run_batch(wf, args.macro_base, args.macro_key.strip(), args.agent_base, args.date, args.dry_run)
        rc = max(rc, code)
    return rc


if __name__ == "__main__":
    raise SystemExit(main())
