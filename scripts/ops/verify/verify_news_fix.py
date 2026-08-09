#!/usr/bin/env python3
"""Final verification after news migration fixes."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    print("## HK mongo + news")
    print(run("geegoo-data", "docker ps --format '{{.Names}} {{.Status}}' | grep mongo"))
    print(
        run(
            "geegoo-data",
            "cd /root/apps/GeeGooData && set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -5",
            timeout=300,
        )
    )
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"TSLA.US","name":{"init":"Tesla"}},{"code":"000858.SZ","name":{"init":"w"}}]}\' '
            '| python3 -c "import sys,json; d=json.load(sys.stdin); print(\'hk_mixed\', len(d), [x.get(\'code\') for x in d[:3]])"',
        )
    )
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"limit":1}\' | python3 -c "import sys,json; d=json.load(sys.stdin); print(type(d).__name__, len(d) if isinstance(d,list) else d)"',
        )
    )

    print("\n## CN news")
    print(run("geegoo-data-cn", "docker ps --format '{{.Names}} {{.Status}}' | grep mongo"))
    print(
        run(
            "geegoo-data-cn",
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"000858.SZ","name":{"init":"五粮液"}}]}\' '
            '| python3 -c "import sys,json; d=json.load(sys.stdin); print(\'cn\', len(d), d[0].get(\'publisher\') if d else None)"',
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
