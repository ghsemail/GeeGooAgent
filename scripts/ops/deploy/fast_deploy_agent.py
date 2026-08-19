#!/usr/bin/env python3
"""Fast agent deploy: git pull + build + restart-all + doctor."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = "/home/ubuntu/.geegoo/geegoo-agent"


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    cmd = (
        f"cd {REMOTE} && git fetch origin main && git reset --hard origin/main && git log -1 --oneline && "
        f"bash start.sh restart-all && sleep 2 && "
        f"curl -sf http://127.0.0.1:3400/health && "
        f"~/.geegoo/bin/geegoo doctor"
    )
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=60)
    _, o, e = c.exec_command(cmd, get_pty=True, timeout=900)
    text = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(text)
    code = o.channel.recv_exit_status()
    c.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())
