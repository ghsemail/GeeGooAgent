#!/usr/bin/env python3
"""Verify Mongo regex for stock name enrichment on GeeGooSignal server."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
REMOTE_PY = r"""
import sys
sys.path.insert(0, '/root/apps/GeeGooSignal/scripts')
from pymongo import MongoClient
from stock_catalog_names import _en_name_query, MONGO_CHINESE_CHAR_REGEX

db = MongoClient('mongodb://127.0.0.1:27017')['Signal_DB']['stock_db']
for markets in (['hk'], ['us'], ['cn']):
    q = _en_name_query(markets)
    print(markets, MONGO_CHINESE_CHAR_REGEX, db.count_documents(q))
"""


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-signal"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )
    deploy_cmd = (
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main"
    )
    _, stdout, stderr = client.exec_command(deploy_cmd)
    print(stdout.read().decode() or stderr.read().decode())
    cmd = f"cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 - <<'PY'\n{REMOTE_PY}\nPY"
    _, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode()
    err = stderr.read().decode()
    client.close()
    print(out or err)
    return 0 if out.strip() else 1


if __name__ == "__main__":
    raise SystemExit(main())
