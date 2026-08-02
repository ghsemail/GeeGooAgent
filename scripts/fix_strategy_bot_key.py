#!/usr/bin/env python3
"""Sync GEEGOO_BOT_SERVICE_API_KEY from GeeGooBot to GeeGooSignal worker .env and restart."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(client: paramiko.SSHClient, cmd: str) -> str:
    _, stdout, stderr = client.exec_command(cmd, timeout=120)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    if out:
        print(out)
    if err:
        print("STDERR:", err)
    return out


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    bot = cfg["targets"]["geegoo-tradingbot"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    bot_client = paramiko.SSHClient()
    bot_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    bot_client.connect(bot["host"], username=bot["user"], password=bot["password"], timeout=30)

    print("=== read GEEGOO_BOT_SERVICE_API_KEY from GeeGooBot ===")
    key = ssh_run(
        bot_client,
        "grep '^GEEGOO_BOT_SERVICE_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-",
    )
    bot_client.close()
    if not key:
        print("ERROR: empty GEEGOO_BOT_SERVICE_API_KEY on GeeGooBot")
        return 1

    sig_client = paramiko.SSHClient()
    sig_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    sig_client.connect(sig["host"], username=sig["user"], password=sig["password"], timeout=30)

    print("=== update GeeGooSignal .env ===")
    ssh_run(
        sig_client,
        "grep -q '^GEEGOO_BOT_SERVICE_API_KEY=' /root/apps/GeeGooSignal/.env "
        f"&& sed -i 's|^GEEGOO_BOT_SERVICE_API_KEY=.*|GEEGOO_BOT_SERVICE_API_KEY={key}|' /root/apps/GeeGooSignal/.env "
        f"|| echo 'GEEGOO_BOT_SERVICE_API_KEY={key}' >> /root/apps/GeeGooSignal/.env",
    )
    ssh_run(sig_client, "grep BOT_SERVICE /root/apps/GeeGooSignal/.env | sed 's/KEY=.*/KEY=***/'")

    print("=== restart GeeGooSignal worker ===")
    ssh_run(
        sig_client,
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    )
    print("=== tail worker log ===")
    import time
    time.sleep(5)
    ssh_run(sig_client, "tail -n 20 /root/apps/GeeGooSignal/worker.out")
    sig_client.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
