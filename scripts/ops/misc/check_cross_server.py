#!/usr/bin/env python3
"""Quick cross-server connectivity checks for CN data routing."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
CN_DATA_ENV = "/home/ubuntu/apps/GeeGooData/.env"


def run(name: str, cmd: str) -> None:
    s = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))["targets"][name]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, _ = c.exec_command(cmd, timeout=60)
    print(f"[{name}] {cmd}\n", o.read().decode())
    c.close()


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    cn_ssh = cfg["targets"]["geegoo-data-cn"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(cn_ssh["host"], username=cn_ssh["user"], password=cn_ssh.get("password"), timeout=60)
    _, o, _ = c.exec_command(
        f"grep '^GEEGOO_DATA_SERVICE_TOKEN=' {CN_DATA_ENV} | head -1 | cut -d= -f2-",
        timeout=30,
    )
    token = o.read().decode().strip()
    c.close()
    if not token:
        raise SystemExit(f"missing GEEGOO_DATA_SERVICE_TOKEN on {CN_DATA_ENV}")

    run("geegoo-bot", 'curl -s -m 10 -o /dev/null -w "%{http_code}" http://82.157.97.76:3300/health; echo')
    run(
        "geegoo-bot",
        f'curl -s -m 15 -H "Authorization: Bearer {token}" -H "Content-Type: application/json" '
        f'-d \'{{"code":"600519.SH"}}\' http://82.157.97.76:3300/v1/market/quote | head -c 200',
    )
    run("geegoo-data-cn", "curl -s -m 5 http://118.195.135.97:3120/health | head -c 100")


if __name__ == "__main__":
    main()
