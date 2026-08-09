#!/usr/bin/env python3
"""Verify TradingData cutover: legacy APIs + backend health."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
ANALYZE_KEY = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"


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
    print("=== GeeGooData legacy ===")
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getSingleAnalysis 2>/dev/null | head -c 1; '
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"AAPL"}]}\' | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d), d[0].get(\'code\') if d else None)"',
        )
    )
    print("=== analyze-api getSingleAnalysis (code field) ===")
    print(
        run(
            "geegoo-signal",
            f'curl -s -X POST http://127.0.0.1:3230/getSingleAnalysis -H "Content-Type: application/json" '
            f'-H "Authorization: Bearer {ANALYZE_KEY}" '
            '-d \'{"code":"AAPL","name":"Apple","prompt_id":"price","save":false}\' | head -c 300',
        )
    )
    print("\n=== backend health (catalog) ===")
    print(
        run(
            "geegoo-signal",
            'curl -s http://127.0.0.1:3210/getBackendHealth -H "Content-Type: application/json" '
            '-d "{}" | python3 -c "import sys,json; d=json.load(sys.stdin); print([(x.get(\'name\'),x.get(\'port\'),x.get(\'status\')) for x in d.get(\'data\',d) if isinstance(d,dict) else d][:10])" 2>/dev/null || '
            'curl -s http://127.0.0.1:3210/getBackendHealth | head -c 600',
        )
    )
    print("\n=== TradingServer :7000 from signal host ===")
    print(run("geegoo-signal", "curl -sf --connect-timeout 5 http://43.134.94.87:7000/health || echo FAIL"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
