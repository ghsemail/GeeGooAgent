#!/usr/bin/env python3
"""Sync GEEGOO_DATA_SERVICE_TOKEN into GeeGooSignal .env and restart analyze-api."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh(target: str, cmd: str, timeout: int = 180) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    code = o.channel.recv_exit_status()
    c.close()
    if code != 0:
        raise RuntimeError(f"{target} exit {code}: {err.strip() or out.strip()}")
    return out


def main() -> None:
    data_token = ssh(
        "geegoo-tradingdata",
        "grep '^GEEGOO_DATA_SERVICE_TOKEN=' /root/apps/GeeGooData/.env | head -1 | cut -d= -f2-",
    ).strip()
    if not data_token:
        raise SystemExit("missing GEEGOO_DATA_SERVICE_TOKEN on GeeGooData host")

    cmds = [
        f"grep -q '^GEEGOO_DATA_SERVICE_TOKEN=' /root/apps/GeeGooSignal/.env || echo 'GEEGOO_DATA_SERVICE_TOKEN=' >> /root/apps/GeeGooSignal/.env",
        f"sed -i 's|^GEEGOO_DATA_SERVICE_TOKEN=.*|GEEGOO_DATA_SERVICE_TOKEN={data_token}|' /root/apps/GeeGooSignal/.env",
        "grep '^GEEGOO_DATA_HTTP_URL=' /root/apps/GeeGooSignal/.env || echo 'GEEGOO_DATA_HTTP_URL=http://47.80.14.120:3300' >> /root/apps/GeeGooSignal/.env",
        "cd /root/apps/GeeGooSignal && bash start.sh restart",
        "sleep 3 && curl -sf http://127.0.0.1:3230/health",
    ]
    for cmd in cmds:
        print(f"\n>>> {cmd}")
        print(ssh("geegoo-tradingsignal", cmd))


if __name__ == "__main__":
    main()
