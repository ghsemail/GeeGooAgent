#!/usr/bin/env python3
"""Update GeeGooAgent on 119.45.16.112 via install.sh."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    t = cfg["targets"]["geegoo-agent"]
    s = t["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        install = t.get(
            "install_cmd",
            "export GEEGOO_SKIP_SETUP=1 DEBIAN_FRONTEND=noninteractive; "
            "curl -fsSL https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | bash",
        )
        cmds = [
            install,
            "sleep 4",
            "curl -sf http://127.0.0.1:3400/health || echo HEALTH_FAIL",
            t.get("verify_cmd", "~/.geegoo/bin/geegoo doctor || true"),
        ]
        for cmd in cmds:
            print(f"\n>>> {cmd[:160]}\n")
            _, o, e = c.exec_command(cmd, timeout=900)
            text = (o.read() + e.read()).decode("utf-8", errors="replace")
            print(text[-4000:])
    finally:
        c.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
