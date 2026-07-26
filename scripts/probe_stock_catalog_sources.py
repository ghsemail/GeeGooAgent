#!/usr/bin/env python3
"""Probe stock catalog data sources on GeeGooSignal server."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
REMOTE_PY = r"""
import json, os, sys
sys.path.insert(0, '/root/apps/GeeGooSignal/scripts')
for line in open('/root/apps/GeeGooSignal/.env'):
    line = line.strip()
    if line and not line.startswith('#') and '=' in line:
        k, v = line.split('=', 1)
        os.environ.setdefault(k, v)
from pymongo import MongoClient
from stock_catalog_names import fetch_stock_info_name

db = MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)['Signal_DB']['stock_db']
out = {
    'futu_host': os.getenv('GEEGOO_FUTU_OPEND_HOST', '127.0.0.1'),
    'futu_port': os.getenv('GEEGOO_FUTU_OPEND_PORT', '11111'),
    'geegoo_data_url': os.getenv('GEEGOO_DATA_HTTP_URL', ''),
    'legacy_stock_info': os.getenv('STOCK_INFO_HTTP_URL', 'http://47.80.14.120:5700'),
    'total': db.count_documents({}),
    'en_filled': db.count_documents({'name.en': {'$exists': True, '$nin': [None, '']}}),
    'zh_hk_filled': db.count_documents({'name.zh_hk': {'$exists': True, '$nin': [None, '']}}),
    'samples': {},
    'name_api': {},
}
for code in ['600519.SH', '00700.HK', 'AAPL.US']:
    doc = db.find_one({'code': code}, {'name': 1, 'market': 1, 'lot_size': 1, 'stock_type': 1})
    out['samples'][code] = doc
out['name_api']['AAPL'] = fetch_stock_info_name('AAPL')
out['name_api']['0700.HK'] = fetch_stock_info_name('0700.HK')
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
        timeout=30,
    )
    cmd = "cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 - <<'PY'\n" + REMOTE_PY + "\nPY"
    _, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode()
    err = stderr.read().decode()
    client.close()
    if out.strip():
        print(out)
        return 0
    print(err or "no output", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
