#!/usr/bin/env python3
"""Diagnose Mongo + GeeGooData on 47.80.14.120."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


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
    cmds = [
        "docker ps -a | grep -i mongo || echo NO_DOCKER_MONGO",
        "ss -lntp | grep 27017 || echo NO_27017",
        "curl -sf http://127.0.0.1:3300/health || echo DATA_API_DOWN",
        'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" -d \'{"limit":1}\' | head -c 400',
        "echo",
        "crontab -l 2>/dev/null | grep -i trading || echo NO_CRON",
        "ps aux | grep -E 'NewsServer|TradingData|start.sh' | grep -v grep | head -10",
    ]
    for cmd in cmds:
        print("$", cmd)
        _, o, e = c.exec_command(cmd)
        print((o.read() + e.read()).decode())
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
