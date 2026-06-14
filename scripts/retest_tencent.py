#!/usr/bin/env python3
"""Retest Tencent analysis endpoints and GeeGoo Agent tools."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import httpx

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from geegoo_agent.config import load_config
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext
from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.clients.market import MarketClient
from geegoo_agent.infra.sandbox import NetworkPolicy

def _resolve_mcp_token(config) -> str:
    return os.environ.get("MCP_TOKEN", "").strip() or (config.mcp_token or "").strip()
CODE, NAME = "00700.HK", "腾讯控股"


def http_probe(config, mcp_token: str) -> list[dict]:
    mk_h = {"Authorization": f"Bearer {config.api_key}", "Content-Type": "application/json"}
    sk_h = {"Authorization": f"Bearer {config.geegoo_api_key}", "Content-Type": "application/json"}
    geegoo = config.geegoo_url.rstrip("/")
    market = config.base_url.rstrip("/")
    rows = []
    tests = [
        ("5700 searchCode", f"{geegoo}/searchCode", sk_h, {"regex": "腾讯", "market": ["HK"]}),
        ("5700 getCurrentPrice", f"{geegoo}/getCurrentPrice", sk_h, {"mcp_token": mcp_token, "code": CODE}),
        (
            "5700 getSinglePromptTemplate",
            f"{geegoo}/getSinglePromptTemplate",
            sk_h,
            {"mcp_token": mcp_token, "type": "tech", "period": "daily"},
        ),
        ("5900 searchCode", f"{market}/searchCode", mk_h, {"regex": "腾讯", "market": ["HK"]}),
        (
            "5900 checkTradingDay",
            f"{market}/checkTradingDay",
            mk_h,
            {"mcp_token": mcp_token, "code": CODE},
        ),
        (
            "5900 getTicker",
            f"{market}/getTicker",
            mk_h,
            {"mcp_token": mcp_token, "code": CODE, "num": 3},
        ),
    ]
    with httpx.Client(timeout=90) as client:
        for label, url, headers, body in tests:
            try:
                r = client.post(url, json=body, headers=headers)
                rows.append(
                    {
                        "label": label,
                        "status": r.status_code,
                        "preview": r.text[:200],
                        "ok": r.status_code == 200,
                    }
                )
            except Exception as exc:
                rows.append({"label": label, "status": "ERR", "preview": str(exc), "ok": False})
    return rows


def tool_chain(config, mcp_token: str) -> list[dict]:
    network = NetworkPolicy(config.sandbox.allowed_hosts)
    market = MarketClient(config.base_url, config.api_key, network, timeout=90, retry_wait_seconds=0)
    geegoo = GeeGooBotClient(config.geegoo_url, config.geegoo_api_key, network, retry_wait_seconds=0)
    reg = register_all_tools(ToolRegistry())
    ctx = ToolContext(
        session_id="retest",
        mcp_token=mcp_token,
        dry_run=False,
        workspace_root=ROOT / "data",
        market_client=market,
        geegoo_bot_client=geegoo,
        project_root=ROOT,
    )
    steps = [
        ("check_trading_day", {"code": CODE}),
        ("search_code", {"regex": "腾讯", "market": ["HK"]}),
        ("get_current_price", {"code": CODE}),
        ("get_ticker", {"code": CODE, "num": 5}),
        ("get_capital_distribution", {"code": CODE}),
        ("get_capital_flow", {"code": CODE, "period": "DAY"}),
        ("get_single_prompt_template", {"type": "tech", "period": "daily"}),
        (
            "get_mcp_analysis",
            {
                "name": NAME,
                "code": CODE,
                "prompt_id": "69ec7035b9ccd3d9befc6c23",
                "period": "daily",
            },
        ),
    ]
    results = []
    for name, args in steps:
        r = reg.execute(ToolCallRequest(name=name, arguments=args), ctx)
        row = {"tool": name, "status": r.status, "summary": r.summary}
        if name == "get_ticker" and r.data:
            items = r.data.get("items") or []
            if items:
                row["price"] = items[-1].get("price")
                row["time"] = items[-1].get("time")
        if name == "get_current_price" and r.data:
            row["price"] = r.data.get("price")
        if name == "get_mcp_analysis" and r.data:
            row["analysis_preview"] = (r.data.get("analysis_result") or "")[:200]
        results.append(row)
    market.close()
    geegoo.close()
    return results


def main() -> int:
    config_path = os.environ.get("GEEGOO_CONFIG", str(ROOT / "config.local.json"))
    config = load_config(config_path)
    mcp_token = _resolve_mcp_token(config)
    if not mcp_token:
        print("Set MCP_TOKEN or config mcp_token", file=sys.stderr)
        return 1
    print("=== HTTP 直连探测 ===")
    for row in http_probe(config, mcp_token):
        mark = "OK" if row["ok"] else "FAIL"
        print(f"[{mark}] {row['label']}: {row['status']}")
        print(f"       {row['preview'][:180]}")
    print("\n=== GeeGoo Agent Tool 链路 ===")
    for row in tool_chain(config, mcp_token):
        mark = "OK" if row["status"] == "ok" else row["status"].upper()
        print(f"[{mark}] {row['tool']}: {row['summary'][:100]}")
        if row.get("price") is not None:
            print(f"       price={row['price']} time={row.get('time','')}")
        if row.get("analysis_preview"):
            print(f"       analysis={row['analysis_preview'][:120]}...")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
