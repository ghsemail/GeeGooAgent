#!/usr/bin/env python3
"""Deploy news enrich fixes to GeeGooSignal + GeeGooData and refresh TSLA cache."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 600) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", "replace")
    client.close()
    return out


def main() -> int:
    print("=== deploy GeeGooSignal ===")
    print(
        run(
            "geegoo-signal",
            "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
        )
    )

    print("\n=== deploy GeeGooData HK ===")
    print(
        run(
            "geegoo-data",
            "cd /root/apps/GeeGooData && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
        )
    )

    print("\n=== refresh news once ===")
    print(
        run(
            "geegoo-data",
            "cd /root/apps/GeeGooData && set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -15",
            timeout=300,
        )
    )

    print("\n=== verify TSLA empty cn count ===")
    print(
        run(
            "geegoo-data",
            "python3 <<'PY'\n"
            "import pymongo\n"
            "c=pymongo.MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)\n"
            "col=c.aidb.news_cache\n"
            "empty=0\n"
            "total=0\n"
            "for r in col.find({'code':'TSLA.US','ts':'2026-08-10'}, {'_id':0,'news':1}):\n"
            "    total += 1\n"
            "    t=(r.get('news') or {}).get('title') or {}\n"
            "    if not (t.get('cn') or '').strip():\n"
            "        empty += 1\n"
            "        print('EMPTY', (t.get('en') or '')[:80])\n"
            "print('empty_cn', empty, 'total', total)\n"
            "PY",
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
