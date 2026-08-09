#!/usr/bin/env python3
"""Set CN data token/URL on GeeGooBot and GeeGooSignal."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
CN_TOKEN_CMD = "grep ^GEEGOO_DATA_SERVICE_TOKEN= /home/ubuntu/apps/GeeGooData/.env | cut -d= -f2-"


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def upsert(remote_dir: str, key: str, value: str) -> str:
    return (
        f"cd {remote_dir} && "
        f"if grep -q '^{key}=' .env 2>/dev/null; then "
        f"sed -i 's|^{key}=.*|{key}={value}|' .env; "
        f"else echo {key}={value} >> .env; fi"
    )


def main() -> int:
    cn_token = run("geegoo-data-cn", CN_TOKEN_CMD).strip()
    if not cn_token:
        raise SystemExit("missing CN data token")
    print("CN token", cn_token[:8] + "...")

    bot_dir = "/home/ubuntu/apps/GeeGooBot"
    sig_dir = "/root/apps/GeeGooSignal"
    for target, rd, restart in [
        ("geegoo-bot", bot_dir, f"cd {bot_dir} && printf '3\\n' | bash start.sh"),
        ("geegoo-tradingsignal", sig_dir, f"cd {sig_dir} && printf '2\\n' | bash start.sh"),
    ]:
        print(f"\n## {target}")
        cmds = [
            upsert(rd, "GEEGOO_DATA_CN_HTTP_URL", "http://82.157.97.76:3300"),
            upsert(rd, "GEEGOO_DATA_CN_SERVICE_TOKEN", cn_token),
            f"grep GEEGOO_DATA_CN / {rd}/.env",
            restart,
        ]
        for cmd in cmds:
            print(run(target, cmd))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
