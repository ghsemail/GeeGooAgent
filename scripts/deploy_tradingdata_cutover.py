#!/usr/bin/env python3
"""Deploy TradingData cutover: restart GeeGooSignal + verify legacy endpoints."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
SIGNAL_KEY = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"


def ssh_run(target: str, cmds: list[str]) -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    for cmd in cmds:
        print(f"[{target}] $ {cmd}")
        _, o, e = c.exec_command(cmd)
        print((o.read() + e.read()).decode())
    c.close()


def main() -> int:
    ssh_run(
        "geegoo-signal",
        [
            "cd /root/apps/GeeGooSignal && bash start.sh restart analyze-api",
            "cd /root/apps/GeeGooSignal && bash start.sh restart catalog-api",
            "sleep 3",
            'curl -sf http://127.0.0.1:3230/health && echo',
            'curl -sf http://127.0.0.1:3210/health && echo',
        ],
    )
    ssh_run(
        "geegoo-data",
        [
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"AAPL"}]}\' | head -c 500',
            "echo",
            'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"limit":2}\' | head -c 500',
            "echo",
            f'curl -s -X POST http://127.0.0.1:3230/getSingleAnalysis -H "Content-Type: application/json" '
            f'-H "Authorization: Bearer {SIGNAL_KEY}" '
            '-d \'{"stock_code":"AAPL"}\' | head -c 500',
            "echo",
        ],
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
