#!/usr/bin/env python3
"""Compare MCP route migration: Python :5700 vs Go :3120 vs Agent catalog."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
PY_MCP = Path(r"D:\Geegoo\TradingBot\mcpAPIServer.py")
GO_HANDLER = Path(r"D:\Geegoo\GeeGooBot\internal\mcp\handler.go")

# Agent bespoke tools -> MCP path (from geegoobot_contract_test + catalog)
BESPOKE = {
    "check_trading_day": "/checkTradingDay",
    "get_current_price": "/getCurrentPrice",
    "get_single_prompt_template": "/getSinglePromptTemplate",
    "get_mcp_analysis": "/getMCPAnalysis",
    "get_capital_flow": "/getCapitalFlow",
    "get_capital_distribution": "/getCapitalDistribution",
    "get_report_bot_codes": "/getReportBotCodes",
    "get_stock_daily_reports": "/getStockDailyReports",
    "list_today_reports": "/getStockDailyReports",
    "get_bot_yesterday_attitude": "/getBotYesterdayAttitude",
    "create_pre_market_report": "/createPreMarketReport",
}


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 30) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    if code != 0 and not out.strip():
        raise RuntimeError(err.strip() or f"exit {code}")
    return out


def parse_python_routes() -> list[str]:
    text = PY_MCP.read_text(encoding="utf-8")
    return sorted(set(re.findall(r"@app\.route\('(/[^']+)'", text)))


def parse_go_routes() -> list[str]:
    text = GO_HANDLER.read_text(encoding="utf-8")
    paths = re.findall(r'HandleFunc\("POST ([^"]+)"', text)
    return sorted(set(paths))


def probe_port(bot_ssh: dict, port: int, path: str, token: str, api_key: str) -> int:
    body = json.dumps({"mcp_token": token, "code": "00700.HK"})
    cmd = (
        f"curl -sS -m 12 -o /dev/null -w '%{{http_code}}' "
        f"-H 'Authorization: Bearer {api_key}' -H 'Content-Type: application/json' "
        f"-d '{body}' http://127.0.0.1:{port}{path}"
    )
    out = ssh_run(bot_ssh, cmd).strip()
    try:
        return int(out)
    except ValueError:
        return -1


def main() -> int:
    py_routes = parse_python_routes()
    go_routes = parse_go_routes()

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    creds = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key',''))\"",
    )
    lines = [ln.strip() for ln in creds.strip().splitlines() if ln.strip()]
    token, api_key = lines[0], lines[1]

    print(f"Python mcpAPIServer routes: {len(py_routes)}")
    print(f"Go mcp-api handler routes:  {len(go_routes)}")
    print(f"Go registered: {', '.join(go_routes)}")
    print()

    migrated = []
    missing_3120 = []
    only_5700 = []

    for path in py_routes:
        code3120 = probe_port(bot_ssh, 3120, path, token, api_key)
        code5700 = probe_port(bot_ssh, 5700, path, token, api_key)
        if code3120 == 200:
            migrated.append((path, code3120, code5700))
        elif code3120 == 404:
            missing_3120.append((path, code3120, code5700))
            if code5700 != 404:
                only_5700.append((path, code3120, code5700))

    print("=== SUMMARY ===")
    print(f"3120 HTTP 200 (migrated & working): {len(migrated)}")
    print(f"3120 HTTP 404 (not on Go mcp-api):   {len(missing_3120)}")
    print(f"Still on 5700 (non-404):           {len(only_5700)}")
    pct = 100.0 * len(migrated) / len(py_routes) if py_routes else 0
    print(f"Migration progress (by live 3120): {pct:.1f}%")
    print()

    print("=== MIGRATED TO 3120 (HTTP 200) ===")
    for path, c31, c57 in migrated:
        print(f"  {path:42} 3120:{c31} 5700:{c57}")

    print()
    print("=== NOT ON 3120 — grouped ===")
    domains = {
        "bot": [],
        "reminder": [],
        "report": [],
        "trading": [],
        "analyst": [],
        "strategy": [],
        "common": [],
        "other": [],
    }
    for path, c31, c57 in missing_3120:
        p = path.lower()
        if "bot" in p and "reminder" not in p:
            domains["bot"].append((path, c57))
        elif "reminder" in p:
            domains["reminder"].append((path, c57))
        elif "report" in p or "stockdaily" in p.replace("/", ""):
            domains["report"].append((path, c57))
        elif any(x in p for x in ("price", "capital", "ticker", "broker", "search", "signal", "tradingday")):
            domains["trading"].append((path, c57))
        elif any(x in p for x in ("prompt", "analysis", "competitor", "etf")):
            domains["analyst"].append((path, c57))
        elif any(x in p for x in ("strategy", "loopback")):
            domains["strategy"].append((path, c57))
        elif "position" in p or "botlog" in p.replace("/", ""):
            domains["common"].append((path, c57))
        else:
            domains["other"].append((path, c57))

    for name, items in domains.items():
        if not items:
            continue
        print(f"\n[{name}] ({len(items)})")
        for path, c57 in items:
            print(f"  {path:42} 3120:404 5700:{c57}")

    print("\n=== BESPOKE AGENT TOOLS (probe 3120) ===")
    for tool, path in sorted(BESPOKE.items()):
        code = probe_port(bot_ssh, 3120, path, token, api_key)
        status = "OK" if code == 200 else f"HTTP {code}"
        print(f"  {tool:28} {path:32} {status}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
