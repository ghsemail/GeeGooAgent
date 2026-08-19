#!/usr/bin/env python3
"""Fast GeeGooAgent deploy: git sync + build + restart-all + doctor."""
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
    t = cfg["targets"]["geegoo-agent"]
    s = t["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    cmds = [
        f"cd {REMOTE} && git fetch origin main && git reset --hard origin/main && git log -1 --oneline",
        f"cd {REMOTE} && bash start.sh restart-all",
        "sleep 2",
        "curl -sf http://127.0.0.1:3400/health",
        f"cd {REMOTE} && ~/.geegoo/bin/geegoo doctor",
    ]
    try:
        for cmd in cmds:
            print(f"\n>>> {cmd}\n")
            _, o, e = c.exec_command(cmd, get_pty=True, timeout=900)
            text = (o.read() + e.read()).decode("utf-8", errors="replace")
            print(text[-6000:] if len(text) > 6000 else text)
            code = o.channel.recv_exit_status()
            if code != 0:
                print(f"FAILED exit {code}: {cmd}")
                return code
    finally:
        c.close()
    print("\n=== GeeGooAgent deploy OK ===")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
