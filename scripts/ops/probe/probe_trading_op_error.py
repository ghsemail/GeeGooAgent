#!/usr/bin/env python3
"""Probe trading_operation web + backend APIs on 146.56.225.252."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["trading-operation"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=22,
        username=ssh["user"],
        password=ssh["password"],
        timeout=20,
    )
    cmds = [
        "curl -sI http://127.0.0.1:8088/ | head -8",
        "docker ps --format '{{.ID}} {{.Ports}} {{.Names}}' | grep 8088 || true",
        "docker ps --format '{{.ID}} {{.Ports}} {{.Names}}' | grep nginx || head -5",
        "grep -r 'op_catalog\\|op_bot\\|op_signal\\|proxy_pass' /root/apps/trading_operation/ 2>/dev/null | head -20",
        "docker exec $(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088/{print $1; exit}') cat /etc/nginx/conf.d/default.conf 2>/dev/null | head -80",
        "curl -s -m 15 -X POST http://127.0.0.1:8088/op_catalog/login -H 'Content-Type: application/json' -d '{\"username\":\"test\",\"password\":\"test\"}' | head -c 500",
        "curl -s -m 15 -X POST http://127.0.0.1:8088/op_bot/getUserList -H 'Content-Type: application/json' -d '{}' | head -c 500",
        "curl -s -m 15 -X POST http://127.0.0.1:8088/op_bot/queryTradingDate -H 'Content-Type: application/json' -d '{}' | head -c 800",
        "curl -s -m 20 -X POST http://127.0.0.1:5800/checkBackendServices -H 'Content-Type: application/json' -d '{}' | head -c 3000",
        "ss -lntp | grep -E ':5800|:3210|:3140|:5600|:8088' || true",
        "tail -40 /root/apps/TradingSignal/admin.out 2>/dev/null || true",
        "tail -20 /root/apps/GeeGooSignal/catalog-api.out 2>/dev/null || true",
    ]
    for cmd in cmds:
        print("\n===", cmd[:90], "===")
        _, o, e = c.exec_command(cmd, timeout=45)
        out = (o.read() + e.read()).decode("utf-8", errors="replace")
        print(out[:4000])
    c.close()


if __name__ == "__main__":
    main()
