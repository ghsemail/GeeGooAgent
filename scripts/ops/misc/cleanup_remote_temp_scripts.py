#!/usr/bin/env python3
"""Remove temporary _* scripts from remote TradingBot/GeeGooBot hosts."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

TARGETS = {
    "geegoo-bot": [
        "find /home/ubuntu/apps/TradingBot/scripts -maxdepth 1 -name '_*' -type f -delete 2>/dev/null; echo tradingbot_scripts_done",
        "find /home/ubuntu/apps/GeeGooBot/scripts -maxdepth 1 -name '_*' -type f -delete 2>/dev/null; echo geegoobot_scripts_done",
        "find /home/ubuntu/apps/TradingBot/scripts -maxdepth 1 -name '*_out.txt' -type f -delete 2>/dev/null; true",
    ],
    "geegoo-agent": [
        "find /home/ubuntu/.geegoo/geegoo-agent/scripts -maxdepth 1 -name '_*' -type f -delete 2>/dev/null; echo agent_scripts_done",
    ],
}


def run(target: str, cmd: str) -> str:
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    for target, cmds in TARGETS.items():
        print(f"=== {target} ===")
        for cmd in cmds:
            print(run(target, cmd).strip())


if __name__ == "__main__":
    main()
