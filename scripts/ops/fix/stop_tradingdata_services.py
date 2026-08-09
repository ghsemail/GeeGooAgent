#!/usr/bin/env python3
"""Stop TradingData Python services after GeeGooData cutover (keep TradingServer)."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
TRADING_DATA = "/root/apps/TradingData"
PORTS = (5500, 5600, 5700, 5800, 5900)
PROCESS_PATTERNS = (
    "AIServer.py",
    "NewsServer.py",
    "StrategyServer.py",
    "USDataServer.py",
    "LLMServer.py",
    "RefreshNews.py",
)


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-tradingdata"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )

    def run(cmd: str) -> str:
        _, o, e = c.exec_command(cmd)
        return (o.read() + e.read()).decode().strip()

    print("=== before ===")
    print(run(f"ss -lntp | grep -E ':({'|'.join(map(str, PORTS))})' || echo NONE"))

    for pat in PROCESS_PATTERNS:
        run(f"pkill -f '{pat}' || true")
        run(f"pkill -9 -f '{pat}' || true")

    # Prevent watchdog cron from restarting NewsServer/RefreshNews.
    run(
        "crontab -l 2>/dev/null | grep -v refresh_news_watchdog | grep -v TradingData || true > /tmp/cron.new; "
        "crontab /tmp/cron.new"
    )

    for port in PORTS:
        run(f"fuser -k {port}/tcp 2>/dev/null || true")
        run(
            f"pid=$(ss -lntp | awk '/:{port} /{{print $NF}}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | head -1); "
            f"[ -n \"$pid\" ] && kill $pid || true"
        )

    print("\n=== after ===")
    print(run(f"ss -lntp | grep -E ':({'|'.join(map(str, PORTS))})' || echo NONE"))
    print(run("curl -sf http://127.0.0.1:3300/health || echo data-api-down"))
    print(run("curl -sf --connect-timeout 3 http://43.134.94.87:7000/health || echo tradingserver-remote-check"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
