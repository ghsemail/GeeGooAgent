#!/usr/bin/env python3
"""Probe A-share stock catalog readability: Mongo + Futu OpenD live fetch."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
REMOTE_PY = r"""
import json
import sys
sys.path.insert(0, '/root/apps/GeeGooSignal/scripts')
from pymongo import MongoClient
from stock_catalog_refresh import fetch_cn_stocks

host = '127.0.0.1'
port = 11111
out = {
    'mongo': {},
    'futu_cn': {'ok': False},
    'samples': [],
}

db = MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)['Signal_DB']['stock_db']
out['mongo']['sh'] = db.count_documents({'market': 'SH'})
out['mongo']['sz'] = db.count_documents({'market': 'SZ'})
for code in ['600519.SH', '000001.SZ', '510300.SH']:
    doc = db.find_one({'code': code}, {'code': 1, 'name': 1, 'market': 1, 'lot_size': 1, 'stock_type': 1, '_id': 0})
    out['samples'].append(doc)

try:
    rows = fetch_cn_stocks(host, port)
    out['futu_cn'] = {
        'ok': True,
        'count': len(rows),
        'sh': sum(1 for r in rows if r.get('market') == 'SH'),
        'sz': sum(1 for r in rows if r.get('market') == 'SZ'),
        'sample': next((r for r in rows if r.get('code') == '600519.SH'), None),
    }
except Exception as e:
    out['futu_cn'] = {'ok': False, 'error': str(e)}

print(json.dumps(out, ensure_ascii=False, default=str))
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
        timeout=60,
    )
    cmd = f"cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 - <<'PY'\n{REMOTE_PY}\nPY"
    _, stdout, stderr = client.exec_command(cmd, timeout=120)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    client.close()
    print(out or err)
    return 0 if out.strip() else 1


if __name__ == "__main__":
    raise SystemExit(main())
