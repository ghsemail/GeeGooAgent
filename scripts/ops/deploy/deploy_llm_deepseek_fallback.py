#!/usr/bin/env python3
"""Deploy GeeGooSignal DeepSeek LLM fallback."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 300) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", "replace")
    client.close()
    return out


def main() -> int:
    print("=== ai_model_db models ===")
    print(
        run(
            "geegoo-signal",
            "cd /root/apps/GeeGooSignal && set -a && source .env && set +a && "
            "python3 -c \"import os,pymongo; c=pymongo.MongoClient(os.environ['GEEGOO_SIGNAL_MONGO_URI'], serverSelectionTimeoutMS=5000); "
            "db=c[os.environ.get('GEEGOO_SIGNAL_MONGO_DB','geegoo_signal')]; "
            "rows=list(db.ai_model_db.find({}, {'name':1,'type':1,'provider':1,'base_url':1,'token':1})); "
            "[print(r.get('type'), r.get('name'), r.get('provider'), (r.get('base_url') or '')[:40], bool(r.get('token'))) for r in rows]\"",
        )
    )

    print("\n=== deploy GeeGooSignal ===")
    print(
        run(
            "geegoo-signal",
            "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
        )
    )

    key = run(
        "geegoo-signal",
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-",
    ).strip()
    payload = json.dumps(
        {"title": "Tesla shares fall after downgrade", "snippet": "Shares dropped"}
    )
    print("\n=== enrichStockNews probe ===")
    print(
        run(
            "geegoo-signal",
            "curl -s -X POST http://127.0.0.1:3230/enrichStockNews "
            "-H 'Content-Type: application/json' "
            f"-H 'Authorization: Bearer {key}' "
            f"-d '{payload}'",
        )[:500]
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
