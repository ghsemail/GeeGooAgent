#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-data-cn"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    cmd = r"""
set -a && source /home/ubuntu/apps/GeeGooData/.env && set +a
echo TOKEN_SET=${GEEGOO_DATA_SERVICE_TOKEN:+yes}
ss -ltn | grep 11111 || echo opend_closed
curl -s -m 15 -X POST http://127.0.0.1:3300/v1/market/quote \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GEEGOO_DATA_SERVICE_TOKEN" \
  -d '{"code":"000858.SZ"}'
echo
curl -s -m 20 -X POST http://127.0.0.1:3300/v1/market/klines \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GEEGOO_DATA_SERVICE_TOKEN" \
  -d '{"code":"000858.SZ","frequency":"daily","limit":3}'
echo
"""
    _, o, e = c.exec_command(cmd, timeout=90)
    print((o.read() + e.read()).decode())
    c.close()


if __name__ == "__main__":
    main()
