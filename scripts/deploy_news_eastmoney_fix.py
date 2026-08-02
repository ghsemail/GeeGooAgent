#!/usr/bin/env python3
"""Deploy latest GeeGooData news fixes to HK (git) and CN (bootstrap tarball)."""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    hk = cfg["targets"]["geegoo-data"]
    ssh = hk["ssh"]
    rd = hk["remote_dir"]

    import paramiko

    print("## ensure mongo HK + CN")
    subprocess.check_call([sys.executable, str(Path(__file__).with_name("ensure_data_mongo.py"))])

    print("## deploy HK git")
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    cmd = (
        f"cd {rd} && git fetch origin main && git reset --hard origin/main && "
        "bash start.sh restart && sleep 3 && bash start.sh status && "
        "set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -10 && "
        'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
        '-d \'{"stock_list":[{"code":"TSLA.US","name":{"init":"Tesla"}}]}\' '
        '| python3 -c "import sys,json; d=json.load(sys.stdin); print(\'hk_us\', len(d))"'
    )
    _, o, e = c.exec_command(cmd, timeout=600)
    print((o.read() + e.read()).decode())
    c.close()

    print("## deploy CN bootstrap")
    subprocess.check_call(
        [sys.executable, str(Path(r"D:\Geegoo\GeeGooData\scripts\bootstrap_cn_node.py"))]
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
