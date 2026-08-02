#!/usr/bin/env python3
"""Recreate HK mongodb with low memory settings after OOM crash loop."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

FIX = r"""
set -e
echo "=== before ==="
free -m | head -3
docker ps -a --format '{{.Names}} {{.Status}}' | grep mongo || true
docker rm -f mongodb 2>/dev/null || true
mkdir -p /data/mongodb
docker run -d --name mongodb --restart unless-stopped \
  --memory=512m --memory-swap=768m \
  -p 127.0.0.1:27017:27017 \
  -v /data/mongodb:/data/db \
  mongo:7 --wiredTigerCacheSizeGB 0.25
sleep 8
echo "=== after ==="
docker ps --format '{{.Names}} {{.Status}}' | grep mongo || echo NO_MONGO
ss -lntp | grep 27017 || echo NO_27017
docker logs mongodb --tail 5 2>&1 || true
curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H 'Content-Type: application/json' -d '{"limit":1}' | head -c 200
echo
curl -s -X POST http://127.0.0.1:3300/getStockNews -H 'Content-Type: application/json' -d '{"stock_list":[{"code":"TSLA.US","name":{"init":"Tesla"}}]}' | python3 -c "import sys,json; d=json.load(sys.stdin); print('tsla', len(d))"
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-data"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(FIX, timeout=180)
    print((o.read() + e.read()).decode())
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
