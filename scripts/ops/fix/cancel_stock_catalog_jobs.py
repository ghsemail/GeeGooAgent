#!/usr/bin/env python3
"""Kill stuck stock catalog refresh processes and clear running jobs."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"

REMOTE_PY = r"""
import os
import signal
import subprocess
from datetime import datetime, timezone

from pymongo import MongoClient

now = datetime.now(timezone.utc)
db = MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)['Signal_DB']

patterns = ['stock_catalog_refresh.py', 'stock_catalog_names', 'enrich_en', 'enrich_zh_hk']
killed = []
try:
    out = subprocess.check_output(['ps', 'aux'], text=True)
except Exception as e:
    out = ''
    print('ps_failed', e)

for line in out.splitlines():
    if 'python' not in line.lower():
        continue
    if not any(p in line for p in patterns):
        continue
    if 'grep' in line:
        continue
    parts = line.split()
    if len(parts) < 2:
        continue
    pid = parts[1]
    try:
        os.kill(int(pid), signal.SIGTERM)
        killed.append(pid)
    except Exception as e:
        print('kill_failed', pid, e)

running = list(db.stock_catalog_jobs.find({'status': 'running'}, {'_id': 1}))
for doc in running:
    job_id = doc['_id']
    db.stock_catalog_jobs.update_one(
        {'_id': job_id},
        {'$set': {
            'status': 'failed',
            'finished_at': now,
            'error': 'cancelled: stuck job cleared manually',
        }},
    )
    db.stock_catalog_jobs.update_one(
        {'_id': job_id, 'steps.status': 'running'},
        {'$set': {
            'steps.$.status': 'failed',
            'steps.$.finished_at': now,
            'steps.$.detail': '已手动取消',
        }},
    )

db.stock_catalog_meta.update_one({'_id': 'default'}, {'$set': {'running_job_id': ''}})
print({'killed_pids': killed, 'cancelled_jobs': [d['_id'] for d in running]})
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
    cmd = f"cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 - <<'PY'\n{REMOTE_PY}\nPY"
    _, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode()
    err = stderr.read().decode()
    client.close()
    print(out or err)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
