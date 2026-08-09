#!/usr/bin/env python3
"""Verify ops web + catalog news proxy after TradingData cutover."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
ADMIN_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"


def run(target: str, cmd: str) -> str:
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
    _, o, e = c.exec_command(cmd)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    print("=== TradingData ports (should be NONE) ===")
    print(
        run(
            "geegoo-tradingdata",
            "ss -lntp | grep -E ':(5500|5600|5700|5800|5900)' || echo ALL_STOPPED",
        )
    )

    print("=== catalog getNewsRefreshLogs via proxy target :3300 ===")
    print(
        run(
            "geegoo-signal",
            f'curl -s -X POST http://127.0.0.1:3210/getNewsRefreshLogs -H "Content-Type: application/json" '
            f'-H "Authorization: Bearer {ADMIN_KEY}" '
            '-d \'{"limit":1}\' | head -c 300',
        )
    )
    print()

    print("=== catalog checkBackendServices (no TradingData ports) ===")
    print(
        run(
            "geegoo-signal",
            f'curl -s -X POST http://127.0.0.1:3210/checkBackendServices -H "Content-Type: application/json" '
            f'-H "Authorization: Bearer {ADMIN_KEY}" '
            '-d "{}" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get(\'data\',{}).get(\'items\',[]) if isinstance(d,dict) else []; '
            "trading=[x for x in items if x.get('stack')=='Trading']; "
            "print('trading_count', len(trading), trading); "
            "print('ports', sorted({x.get('port') for x in items}))\"",
        )
    )

    print("=== ops web :8088 ===")
    print(run("geegoo-tradingsignal", "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8088/"))
    print()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
