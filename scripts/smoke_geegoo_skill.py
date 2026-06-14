# coding=utf-8
"""Smoke test geegoo skill MCP endpoints @ 5700."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import requests

GEEGOO_CONFIG = Path(r"C:/Users/ghsemail/.cursor/skills/geegoo/config.json")


def main() -> int:
    cfg = json.loads(GEEGOO_CONFIG.read_text(encoding="utf-8"))
    base = cfg["base_url"].rstrip("/")
    headers = {
        "Authorization": f"Bearer {cfg['api_key']}",
        "Content-Type": "application/json",
    }
    token = cfg["mcp_token"]
    passed = 0
    total = 0

    print(f"Base: {base}")
    print("-" * 72)

    def run(name: str, path: str, body: dict | None = None, *, need_token: bool = False) -> bool:
        nonlocal passed, total
        total += 1
        payload = dict(body or {})
        if need_token:
            payload["mcp_token"] = token
        try:
            resp = requests.post(f"{base}{path}", json=payload, headers=headers, timeout=60)
            data = resp.json() if resp.headers.get("content-type", "").startswith("application/json") else {}
            code = data.get("code") if isinstance(data, dict) else None
            ok = resp.status_code == 200
            summary = ""

            if path == "/searchCode":
                if isinstance(data, list):
                    summary = f"{len(data)} hits"
                    ok = ok and len(data) > 0
                elif isinstance(data, dict):
                    items = data.get("data")
                    summary = f"{len(items) if isinstance(items, list) else 0} hits"
                    ok = ok and items is not None
            elif path == "/getIndexSignalForSkill":
                if isinstance(data, list):
                    summary = f"{len(data)} signals"
                    ok = ok and len(data) > 0
                elif isinstance(data, dict):
                    items = data.get("data", [])
                    summary = f"{len(items)} signals"
                    ok = ok and code == 100
            elif path == "/getReportBotCodes":
                items = data.get("data", [])
                summary = f"{len(items)} report target(s)"
                ok = ok and code == 100
            elif path == "/checkTradingDay":
                d = data.get("data") or {}
                summary = f"is_trading_day={d.get('is_trading_day')}"
                ok = ok and code == 100
            elif path == "/getCapitalFlow":
                items = data.get("data", [])
                summary = f"{len(items)} flow rows"
                ok = ok and code == 100
            elif path == "/getMCPAnalysis":
                d = data.get("data")
                summary = "has analysis" if d else "empty"
                ok = ok and code == 100 and bool(d)
            else:
                ok = ok and code == 100

            status = "PASS" if ok else "FAIL"
            print(f"{status}  {name:28}  HTTP {resp.status_code}  code={code}  {summary}")
            if not ok:
                msg = data.get("message") or data.get("error") if isinstance(data, dict) else resp.text[:120]
                print(f"       -> {msg}")
            if ok:
                passed += 1
            return ok
        except Exception as exc:
            print(f"FAIL  {name:28}  {type(exc).__name__}: {exc}")
            return False

    tests = [
        ("公共-搜码", "/searchCode", {"regex": "腾讯", "market": ["HK"]}, False),
        ("行情-信号列表", "/getIndexSignalForSkill", {}, False),
        ("Workflow-交易日", "/checkTradingDay", {"code": "00700.HK"}, True),
        ("Workflow-报告标的", "/getReportBotCodes", {}, True),
        ("Workflow-资金流向", "/getCapitalFlow", {"code": "00700.HK", "period": "DAY"}, True),
        (
            "分析-MCP(轻量)",
            "/getMCPAnalysis",
            {
                "name": "腾讯控股",
                "code": "00700.HK",
                "prompt_id": cfg["pre_market"]["prompt_id"],
                "period": "hourly",
                "language": "cn",
            },
            True,
        ),
    ]
    for item in tests:
        name, path, body, need_token = item
        run(name, path, body, need_token=need_token)

    print("-" * 72)
    print(f"合计: {passed}/{total} 通过")
    return 0 if passed == total else 1


if __name__ == "__main__":
    sys.exit(main())
