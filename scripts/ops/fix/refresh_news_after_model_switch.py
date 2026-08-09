#!/usr/bin/env python3
import json
import sys
import time
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(host, user, password, cmd, timeout=900):
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(host, username=user, password=password, timeout=30)
    _, stdout, stderr = c.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    c.close()
    if out:
        print(out)
    if err:
        print("STDERR:", err[:800])
    return out


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    data = cfg["targets"]["geegoo-tradingdata"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    print("=== deploy GeeGooData ===")
    ssh_run(
        data["host"], data["user"], data["password"],
        "cd /root/apps/GeeGooData && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
        timeout=120,
    )
    time.sleep(5)

    key = ssh_run(
        sig["host"], sig["user"], sig["password"],
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-",
        timeout=30,
    ).strip()

    print("=== enrich smoke ===")
    ssh_run(
        sig["host"], sig["user"], sig["password"],
        "curl -sf -X POST http://127.0.0.1:3230/enrichStockNews "
        "-H 'Content-Type: application/json' "
        f"-H 'Authorization: Bearer {key}' "
        "-d '{\"title\":\"Apple shares surge\",\"snippet\":\"beat estimates\"}'",
        timeout=120,
    )

    print("=== news-worker -once ===")
    ssh_run(
        data["host"], data["user"], data["password"],
        "cd /root/apps/GeeGooData && set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -25",
        timeout=900,
    )

    py = r"""
import sys,json
d=json.load(sys.stdin)
n=d[0] if d else {}
t=n.get('title',{})
s=n.get('summary',{})
cn=t.get('cn') or ''
sc=s.get('cn') or ''
print('title.cn', cn[:100])
print('title_zh', any('\u4e00'<=c<='\u9fff' for c in cn))
print('summary_zh', any('\u4e00'<=c<='\u9fff' for c in sc))
"""

    for code in ("TSLA.US", "AAPL.US", "00700.HK"):
        payload = json.dumps(
            {"stock_list": [{"code": code, "name": {"init": code}}], "language": "cn"},
            ensure_ascii=False,
        )
        print(f"=== verify {code} ===")
        ssh_run(
            data["host"], data["user"], data["password"],
            "curl -sf -X POST http://127.0.0.1:3300/getStockNews "
            "-H 'Content-Type: application/json' "
            f"-d {json.dumps(payload)} | python3 -c {json.dumps(py)}",
            timeout=60,
        )
    return 0


if __name__ == "__main__":
    sys.exit(main())
