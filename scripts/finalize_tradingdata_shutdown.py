#!/usr/bin/env python3
"""Finalize TradingData shutdown: disable watchdog cron, fix mongo, stop Python."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
PORTS = (5500, 5600, 5700, 5800, 5900)
PATTERNS = (
    "AIServer.py",
    "NewsServer.py",
    "StrategyServer.py",
    "USDataServer.py",
    "LLMServer.py",
    "RefreshNews.py",
)


def run(c: paramiko.SSHClient, cmd: str) -> str:
    print("$", cmd)
    _, o, e = c.exec_command(cmd)
    out = (o.read() + e.read()).decode()
    print(out)
    return out


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-data"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )

    # Disable TradingData refresh watchdog cron (restarts NewsServer every 5 min).
    run(
        c,
        "crontab -l 2>/dev/null | grep -v refresh_news_watchdog | grep -v TradingData || true > /tmp/cron.new; "
        "crontab /tmp/cron.new; crontab -l 2>/dev/null | grep -i trading || echo TRADING_CRON_REMOVED",
    )

    # Restart MongoDB container (was crash-looping).
    run(c, "docker restart mongodb || docker start mongodb || true")
    run(c, "sleep 5; docker ps -a | grep mongo; ss -lntp | grep 27017 || echo NO_27017")

    for pat in PATTERNS:
        run(c, f"pkill -9 -f '{pat}' || true")
    for port in PORTS:
        run(c, f"fuser -k {port}/tcp 2>/dev/null || true")

    run(c, f"ss -lntp | grep -E ':({'|'.join(map(str, PORTS))})' || echo ALL_TRADINGDATA_STOPPED")
    run(c, "curl -sf http://127.0.0.1:3300/health && echo")
    run(
        c,
        'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" '
        '-d \'{"limit":1}\' | head -c 200',
    )
    print()
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
