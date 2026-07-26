#!/usr/bin/env python3
"""Ensure GeeGooSignal stock-catalog Python venv has futu + pymongo."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(client: paramiko.SSHClient, cmd: str) -> str:
    _, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode()
    err = stderr.read().decode()
    return (out + err).strip()


def main() -> int:
    with DEPLOY_CFG.open(encoding="utf-8") as f:
        ssh_cfg = json.load(f)["targets"]["geegoo-tradingsignal"]["ssh"]

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )

    cmds = [
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main",
        "cd /root/apps/GeeGooSignal && chmod +x start.sh",
        "cd /root/apps/GeeGooSignal && bash start.sh restart",
        "cd /root/apps/GeeGooSignal && test -x .venv-stock-catalog/bin/python3 || (python3 -m venv .venv-stock-catalog && .venv-stock-catalog/bin/pip install -q --upgrade pip && .venv-stock-catalog/bin/pip install -q -r scripts/requirements-stock-catalog.txt)",
        "cd /root/apps/GeeGooSignal && grep -q '^GEEGOO_STOCK_CATALOG_REFRESH_COMMAND=' .env 2>/dev/null && sed -i 's|^GEEGOO_STOCK_CATALOG_REFRESH_COMMAND=.*|GEEGOO_STOCK_CATALOG_REFRESH_COMMAND=/root/apps/GeeGooSignal/.venv-stock-catalog/bin/python3|' .env || echo 'GEEGOO_STOCK_CATALOG_REFRESH_COMMAND=/root/apps/GeeGooSignal/.venv-stock-catalog/bin/python3' >> .env",
        "cd /root/apps/GeeGooSignal && bash start.sh restart",
        "cd /root/apps/GeeGooSignal && .venv-stock-catalog/bin/python3 -c 'import futu, pymongo; print(\"deps_ok\")'",
        "grep -E '^GEEGOO_STOCK_CATALOG_REFRESH_COMMAND=' /root/apps/GeeGooSignal/.env || true",
    ]
    for cmd in cmds:
        print(f">>> {cmd}")
        print(run(client, cmd))
        print()

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
