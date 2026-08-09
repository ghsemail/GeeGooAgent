#!/usr/bin/env python3
"""Probe all trading_app / trading_operation backend APIs for connectivity."""
from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any

# Keys from client server_url.dart (must match production)
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
CATALOG_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
SIGNAL_KEY = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"
ANALYZE_KEY = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"

PROBE_USER = "000000000000000000000001"
PROBE_BOT = "000000000000000000000099"


@dataclass
class Result:
    group: str
    name: str
    url: str
    status: str  # OK | WARN | FAIL
    detail: str


results: list[Result] = []


def call(
    method: str,
    url: str,
    *,
    token: str | None = None,
    body: dict | None = None,
    timeout: int = 25,
) -> tuple[int | None, Any, str]:
    data = None
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        data = json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read()
            code = resp.status
            try:
                parsed = json.loads(raw.decode("utf-8", errors="replace"))
            except json.JSONDecodeError:
                parsed = raw.decode("utf-8", errors="replace")[:200]
            return code, parsed, ""
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            parsed = json.loads(raw.decode("utf-8", errors="replace"))
        except json.JSONDecodeError:
            parsed = raw.decode("utf-8", errors="replace")[:200]
        return e.code, parsed, ""
    except Exception as e:
        return None, None, str(e)


def probe(
    group: str,
    name: str,
    method: str,
    base: str,
    path: str,
    *,
    token: str | None = None,
    body: dict | None = None,
    timeout: int = 25,
    ok_if: str = "http200",
) -> None:
    base = base.rstrip("/")
    path = path if path.startswith("/") else f"/{path}"
    url = f"{base}{path}"
    code, data, err = call(method, url, token=token, body=body, timeout=timeout)
    if code is None:
        results.append(Result(group, name, url, "FAIL", err[:120]))
        print(f"  [FAIL] {name}: {err[:80]}", flush=True)
        return

    detail = ""
    if isinstance(data, dict):
        if "code" in data:
            detail = f"http={code} code={data.get('code')} msg={str(data.get('message',''))[:60]}"
        elif "error" in data:
            detail = f"http={code} error={str(data.get('error'))[:60]}"
        else:
            detail = f"http={code} keys={list(data.keys())[:6]}"
    elif isinstance(data, list):
        detail = f"http={code} list_len={len(data)}"
    else:
        detail = f"http={code} {str(data)[:60]}"

    status = "OK"
    if code >= 500:
        status = "FAIL"
    elif code == 404:
        status = "FAIL"
    elif code == 401:
        status = "FAIL"
    elif ok_if == "http200" and code != 200:
        status = "WARN"
    elif isinstance(data, dict) and data.get("code") not in (None, 100) and code == 200:
        # business error but reachable
        status = "WARN"

    results.append(Result(group, name, url, status, detail))
    mark = {"OK": "OK", "WARN": "WARN", "FAIL": "FAIL"}[status]
    print(f"  [{mark}] {name}: {detail}", flush=True)


def section(title: str) -> None:
    print(f"\n=== {title} ===", flush=True)


def main() -> int:
    t0 = time.time()

    section("Health")
    for name, url in [
        ("GeeGooBot app-api", "http://118.195.135.97:3100/health"),
        ("GeeGooBot agent-api", "http://118.195.135.97:3110/health"),
        ("GeeGooBot service-api", "http://118.195.135.97:3140/health"),
        ("GeeGooSignal catalog", "http://146.56.225.252:3210/health"),
        ("GeeGooSignal signal", "http://146.56.225.252:3200/health"),
        ("GeeGooSignal analyze", "http://146.56.225.252:3230/health"),
        ("GeeGooData HK", "http://47.80.14.120:3300/health"),
        ("GeeGooData CN", "http://82.157.97.76:3300/health"),
        ("TradingServer", "http://43.134.94.87:7000/health"),
        ("GeeGooAgent runtime", "http://119.45.16.112:3400/health"),
    ]:
        code, data, err = call("GET", url, timeout=10)
        if code == 200:
            results.append(Result("health", name, url, "OK", "http=200"))
            print(f"  [OK] {name}", flush=True)
        else:
            results.append(Result("health", name, url, "FAIL", err or f"http={code}"))
            print(f"  [FAIL] {name}: {err or code}", flush=True)

    code, _, err = call("GET", "http://146.56.225.252:8088/", timeout=10)
    if code in (200, 301, 302, 304):
        results.append(Result("health", "Ops nginx :8088", "http://146.56.225.252:8088/", "OK", f"http={code}"))
        print(f"  [OK] Ops nginx :8088 (http={code})", flush=True)
    else:
        results.append(Result("health", "Ops nginx :8088", "http://146.56.225.252:8088/", "FAIL", err or f"http={code}"))
        print(f"  [FAIL] Ops nginx :8088", flush=True)

    BOT = "http://118.195.135.97:3100"
    AGENT = "http://118.195.135.97:3110"
    SERVICE = "http://118.195.135.97:3140"
    CATALOG = "http://146.56.225.252:3210"
    SIGNAL = "http://146.56.225.252:3200"
    ANALYZE = "http://146.56.225.252:3230"
    DATA_HK = "http://47.80.14.120:3300"
    DATA_CN = "http://82.157.97.76:3300"
    OPS = "http://146.56.225.252:8088"

    # trading_app — GeeGooBot app-api
    section("trading_app → GeeGooBot app-api :3100")
    app_reads = [
        ("getUserStock", {"user_id": PROBE_USER, "language": "cn"}),
        ("getUserInfo", {"user_id": PROBE_USER}),
        ("getNewsStock", {"user_id": PROBE_USER}),
        ("getCurrentPrice", {"code": "TSLA.US"}),
        ("usedCheck", {"user_id": PROBE_USER, "name": "test"}),
        ("hdgCount", {"user_id": PROBE_USER}),
        ("queryVersion", {"name": "slot"}),
    ]
    for path, body in app_reads:
        probe("app", path, "POST", BOT, path, token=BOT_KEY, body=body)

    # login with bad creds — expect business error not connection fail
    probe("app", "login", "POST", BOT, "login", token=BOT_KEY, body={"username": "probe", "password": "probe"})

    section("trading_app → GeeGooBot agent-api :3110")
    for path, body in [
        ("getUserAgents", {"user_id": PROBE_USER}),
        ("getAgentSessionMessages", {"user_id": PROBE_USER, "session_id": "probe"}),
    ]:
        probe("agent", path, "POST", AGENT, path, token=BOT_KEY, body=body)

    section("trading_app → GeeGooBot service-api :3140")
    probe("service", "reports/daily", "POST", SERVICE, "reports/daily", token=BOT_KEY,
          body={"user_id": PROBE_USER, "limit_per_phase": 1})

    section("trading_app → GeeGooSignal catalog :3210")
    for path, body in [
        ("getIndexSignal", {}),
        ("getCustomSignal", {}),
        ("getSignalCombination", {}),
        ("getAllIndexSignalId", {}),
        ("getAISignal", {"code_list": ["TSLA.US"], "month": "2026-08"}),
        ("queryVersion", {"name": "slot"}),
        ("getSinglePromptTemplate", {"type": "single", "user_id": PROBE_USER}),
        ("getAttitudePromptList", {"user_id": PROBE_USER}),
    ]:
        probe("catalog", path, "POST", CATALOG, path, token=CATALOG_KEY, body=body)

    section("trading_app → GeeGooSignal signal-api :3200")
    for path, body in [
        ("searchCode", {"regex": "TSLA"}),
        ("getDashboardKline", {"code": "TSLA.US", "language": "cn"}),
        ("getSupportingPrice", {"code": "TSLA.US"}),
        ("getDashboardSignal", {"code": "TSLA.US", "frequency": "1d", "type": "stock", "signal_index_list": [], "language": "cn"}),
    ]:
        probe("signal", path, "POST", SIGNAL, path, token=SIGNAL_KEY, body=body)

    section("trading_app → GeeGooSignal analyze-api :3230")
    for path, body in [
        ("getSingleAnalysisHistory", {"user_id": PROBE_USER, "type": "single"}),
        ("getSingleAnalysis", {"user_id": PROBE_USER, "name": "t", "code": "TSLA.US", "prompt_id": "x", "period": "1d", "language": "cn"}),
    ]:
        probe("analyze", path, "POST", ANALYZE, path, token=ANALYZE_KEY, body=body, timeout=60)

    section("trading_app → GeeGooData news")
    probe("data-hk", "getStockNews US/HK", "POST", DATA_HK, "getStockNews", body={
        "stock_list": [{"code": "TSLA.US", "name": {"init": "Tesla"}}]})
    probe("data-cn", "getStockNews CN", "POST", DATA_CN, "getStockNews", body={
        "stock_list": [{"code": "000858.SZ", "name": {"init": "五粮液"}}]})

    # trading_operation
    section("trading_operation → catalog-api :3210")
    for path, body in [
        ("getIndexSignal", {}),
        ("getCustomSignal", {}),
        ("getCustomStrategyDefinitions", {}),
        ("getSignalCombination", {}),
        ("getVersion", {}),
        ("getModel", {}),
        ("queryModel", {}),
        ("checkBackendServices", {}),
        ("checkAuxiliaryServices", {}),
        ("getSinglePromptTemplate", {"type": "single", "user_id": PROBE_USER}),
        ("getLLMResult", {"prompt": "ping", "dict": {}}),
        ("searchCode", {"regex": "TSLA"}),
        ("loopBackStrategy", {"code": "TSLA.US"}),
        ("getNewsRefreshLogs", {"limit": 1}),
        ("getStrategyGenerationLogs", {"limit": 1}),
    ]:
        probe("ops-catalog", path, "POST", CATALOG, path, token=CATALOG_KEY, body=body, timeout=120)

    section("trading_operation → GeeGooBot app-api :3100")
    for path, body in [
        ("getUserList", {}),
        ("getKeyList", {}),
        ("queryTradingDate", {}),
    ]:
        probe("ops-bot", path, "POST", BOT, path, token=BOT_KEY, body=body)

    section("trading_operation → signal-api :3200 (v1 stocks)")
    for name, method, path, body in [
        ("GET v1/stocks/stats", "GET", "/v1/stocks/stats", None),
        ("GET v1/stocks/schedule", "GET", "/v1/stocks/schedule", None),
        ("GET v1/stocks/jobs", "GET", "/v1/stocks/jobs", None),
        ("POST searchCode", "POST", "/searchCode", {"regex": "TSLA"}),
    ]:
        probe("ops-signal", name, method, SIGNAL, path, token=SIGNAL_KEY, body=body)

    section("trading_operation → agent BFF :3110/op_agent")
    for name, path in [
        ("GET v1/tools", "/op_agent/v1/tools"),
        ("GET v1/doctor", "/op_agent/v1/doctor?skip_connectivity=true"),
        ("GET v1/metrics/overview", "/op_agent/v1/metrics/overview"),
        ("GET v1/memory/status", "/op_agent/v1/memory/status"),
    ]:
        probe("ops-agent", name, "GET", AGENT, path, token=BOT_KEY)

    section("trading_operation → nginx proxies :8088")
    for name, path in [
        ("op_catalog health", "/op_catalog/health"),
        ("op_catalog getIndexSignal", "/op_catalog/getIndexSignal"),
        ("op_bot getUserList", "/op_bot/getUserList"),
        ("op_signal searchCode", "/op_signal/searchCode"),
    ]:
        method = "GET" if path.endswith("health") else "POST"
        body = {} if method == "POST" else None
        token = CATALOG_KEY if "catalog" in path or "signal" in path and "op_signal" in path else BOT_KEY
        if "op_signal" in path:
            token = SIGNAL_KEY
            body = {"regex": "TSLA"}
        probe("ops-proxy", name, method, OPS, path, token=token, body=body)

    # Summary
    print("\n" + "=" * 60, flush=True)
    ok = sum(1 for r in results if r.status == "OK")
    warn = sum(1 for r in results if r.status == "WARN")
    fail = sum(1 for r in results if r.status == "FAIL")
    print(f"Done in {time.time()-t0:.1f}s — OK={ok} WARN={warn} FAIL={fail}", flush=True)
    if fail:
        print("\n--- FAILURES ---", flush=True)
        for r in results:
            if r.status == "FAIL":
                print(f"  [{r.group}] {r.name}: {r.detail}", flush=True)
    if warn:
        print("\n--- WARNINGS (reachable, business/empty) ---", flush=True)
        for r in results:
            if r.status == "WARN":
                print(f"  [{r.group}] {r.name}: {r.detail}", flush=True)

    out = Path(__file__).with_name("probe_client_apis_result.json")
    out.write_text(json.dumps([r.__dict__ for r in results], ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"\nFull results: {out}", flush=True)
    return 1 if fail else 0


if __name__ == "__main__":
    raise SystemExit(main())
