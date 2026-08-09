#!/usr/bin/env python3
"""Mark stale stock catalog job as failed and clear running_job_id."""
import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
JOB_ID = sys.argv[1] if len(sys.argv) > 1 else "job_1785056903750000930"

cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
target = cfg["targets"]["geegoo-signal"]
ssh = target["ssh"]

remote_py = f"""
from pymongo import MongoClient
from datetime import datetime, timezone
c = MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)
db = c['Signal_DB']
now = datetime.now(timezone.utc)
db.stock_catalog_jobs.update_one(
    {{'_id': '{JOB_ID}'}},
    {{'$set': {{'status': 'failed', 'finished_at': now, 'error': 'interrupted by service restart'}}}},
)
db.stock_catalog_meta.update_one({{'_id': 'default'}}, {{'$set': {{'running_job_id': ''}}}})
doc = db.stock_catalog_jobs.find_one({{'_id': '{JOB_ID}'}})
print(doc)
"""

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(
    hostname=ssh["host"],
    port=int(ssh.get("port", 22)),
    username=ssh["user"],
    password=ssh.get("password"),
    timeout=30,
)
cmd = f"cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 - <<'PY'\n{remote_py}\nPY"
_, stdout, stderr = client.exec_command(cmd)
out = stdout.read().decode()
err = stderr.read().decode()
client.close()
print(out or err)
