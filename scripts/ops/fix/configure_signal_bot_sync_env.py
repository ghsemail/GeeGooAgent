#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh(target: str, cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, e = c.exec_command(cmd, timeout=120)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    if err.strip():
        print("stderr:", err.strip())
    return out


def main() -> None:
    key_line = ssh("geegoo-bot", "grep -E '^GEEGOO_BOT_AGENT_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env | head -1")
    bot_key = key_line.strip().split("=", 1)[1].strip().strip('"')
    envf = "/root/apps/GeeGooSignal/.env"
    ssh("geegoo-signal", f"test -f {envf} || touch {envf}")
    for line in (
        "GEEGOO_BOT_AGENT_API_URL=http://118.195.135.97:3110",
        f"GEEGOO_BOT_AGENT_API_KEY={bot_key}",
    ):
        k = line.split("=", 1)[0]
        ssh(
            "geegoo-signal",
            f"grep -q '^{k}=' {envf} && sed -i 's|^{k}=.*|{line}|' {envf} || echo '{line}' >> {envf}",
        )
    print(ssh("geegoo-signal", f"grep GEEGOO_BOT_AGENT {envf}"))
    print(ssh("geegoo-signal", "cd /root/apps/GeeGooSignal && bash start.sh restart catalog-api"))
    print(ssh("geegoo-signal", "tail -3 /root/apps/GeeGooSignal/catalog-api.out"))


if __name__ == "__main__":
    main()
