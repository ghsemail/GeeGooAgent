#!/usr/bin/env python3
"""Rebuild and restart agent-runtime on agent server."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REPO = "/home/ubuntu/.geegoo/geegoo-agent"


def run(c: paramiko.SSHClient, cmd: str, timeout: int = 900) -> int:
    print(f"\n>>> {cmd}\n")
    _, o, e = c.exec_command(cmd, timeout=timeout)
    text = (o.read() + e.read()).decode("utf-8", errors="replace")
    if text.strip():
        print(text[-5000:])
    return o.channel.recv_exit_status()


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        steps = [
            f"cd {REPO} && git fetch origin main && git reset --hard origin/main",
            f"cd {REPO} && git log -1 --oneline",
            f"cd {REPO} && bash start.sh restart-runtime",
            "sleep 2",
            "curl -sf http://127.0.0.1:3400/health",
            "curl -s -o /dev/null -w 'tools:%{http_code}\\n' http://127.0.0.1:3400/v1/tools",
            f"tail -n 5 {REPO}/agent-runtime.out",
        ]
        for cmd in steps:
            code = run(c, cmd)
            if code != 0 and "curl" not in cmd and "tools:" not in cmd:
                return code
    finally:
        c.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
