#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))


def run(target: str, cmd: str) -> str:
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
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    print("=== restart catalog-api ===")
    print(run("geegoo-signal", "cd /root/apps/GeeGooSignal && printf '4\\n' | bash start.sh | tail -5"))

    print("\n=== HK getStockNews ===")
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"TSLA.US","name":{"init":"Tesla"}}]}\' '
            "| python3 -c \"import sys,json; d=json.load(sys.stdin); n=d[0] if d else {}; t=n.get('title',{}); print('count',len(d),'attitude',n.get('attitude'),'en',t.get('en','')[:60],'cn',t.get('cn',''))\"",
        )
    )

    print("\n=== deleteNewsRefreshLogs ===")
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/deleteNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"run_id":"__nonexistent__"}\'',
        )
    )

    print("\n=== catalog delete proxy ===")
    print(
        run(
            "geegoo-signal",
            'curl -s -X POST http://127.0.0.1:3210/deleteNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"run_id":"__nonexistent__"}\'',
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
