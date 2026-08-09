#!/usr/bin/env python3
"""Probe getUserStockTrend response for dashboard debugging."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    script = r"""
set -e
cd /home/ubuntu/apps/GeeGooBot
set -a; source .env; set +a
UID=$(python3 -c "
from pymongo import MongoClient
import os
c=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])
db=c[os.environ['GEEGOO_BOT_MONGO_DB']]
doc=db.user_security.find_one({'code': {'\$exists': True}})
print(doc['user_id'] if doc else '')
")
echo USER=$UID
BODY=$(python3 -c "import json; print(json.dumps({'user_id':'$UID','type':'flag','frequency':'5m','signal_index_list':[],'language':'cn'}))")
curl -s -X POST http://127.0.0.1:3100/getUserStockTrend \
  -H "Authorization: Bearer $GEEGOO_BOT_APP_API_KEY" \
  -H "Content-Type: application/json" \
  -d "$BODY" > /tmp/trend.json
python3 - <<'PY'
import json
raw=open('/tmp/trend.json').read()
try:
    data=json.loads(raw)
except Exception as e:
    print('INVALID JSON', e)
    print(raw[:500])
    raise SystemExit(1)
if isinstance(data, dict):
    print('ENVELOPE', data)
    raise SystemExit(0)
print('count', len(data))
for i, row in enumerate(data[:3]):
    print('---', i, '---')
    for k,v in row.items():
        print(k, type(v).__name__, repr(v)[:120])
PY
"""
    _, stdout, stderr = client.exec_command(f"bash -lc {json.dumps(script)}", timeout=90)
    out = (stdout.read() + stderr.read()).decode("utf-8", "replace")
    print(out)
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
