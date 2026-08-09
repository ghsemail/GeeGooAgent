#!/usr/bin/env python3
"""Check whether legacy TradingBot/TradingData/TradingSignal are still running."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

CHECKS = [
    (
        "TradingBot",
        "geegoo-tradingbot",
        r"""
echo '=== legacy ports 5500-5800,6100 ==='
ss -lntp 2>/dev/null | grep -E ':(5500|5600|5700|5800|6100)' || echo LEGACY_PORTS_DOWN
echo '=== legacy processes ==='
ps aux | grep -E 'TradingBot|apiServer|mcpAPIServer|utilityServer|agentServer|reportServer' | grep -v grep | head -8 || echo NO_LEGACY_PROC
echo '=== GeeGooBot ports 31xx ==='
ss -lntp 2>/dev/null | grep -E ':(3100|3110|3120|3140)' || echo GEEGOO_PORTS_DOWN
""",
    ),
    (
        "TradingSignal",
        "geegoo-tradingsignal",
        r"""
echo '=== legacy ports 5600-6200 ==='
ss -lntp 2>/dev/null | grep -E ':(5600|5700|5900|6100|6200)' || echo LEGACY_PORTS_DOWN
echo '=== legacy processes ==='
ps aux | grep -E 'TradingSignal|signalAPIServer|promptServer|SKILLServer|tempAPIServer|analyzeAPIServer' | grep -v grep | head -8 || echo NO_LEGACY_PROC
echo '=== GeeGooSignal ports 32xx + nginx 8088 ==='
ss -lntp 2>/dev/null | grep -E ':(3200|3210|3230|8088)' || echo GEEGOO_PORTS_DOWN
""",
    ),
    (
        "TradingData",
        "geegoo-tradingdata",
        r"""
echo '=== legacy ports 5500-6200 ==='
ss -lntp 2>/dev/null | grep -E ':(5500|5600|5700|5800|5900|6100|6200)' || echo LEGACY_PORTS_DOWN
echo '=== legacy processes ==='
ps aux | grep -E 'TradingData|NewsServer|AIServer|analyzeServer|refresh_news' | grep -v grep | head -10 || echo NO_LEGACY_PROC
echo '=== GeeGooData port 3300 ==='
ss -lntp 2>/dev/null | grep -E ':3300' || echo GEEGOO_PORTS_DOWN
""",
    ),
]


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    for label, key, cmd in CHECKS:
        ssh = cfg["targets"][key]["ssh"]
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        client.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
        _, stdout, stderr = client.exec_command(cmd.strip(), timeout=30)
        out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace").strip()
        print(f"===== {label} ({ssh['host']}) =====")
        print(out)
        print()
        client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
