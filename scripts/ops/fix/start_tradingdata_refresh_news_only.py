#!/usr/bin/env python3
"""Start only TradingData RefreshNews scheduler (no HTTP ports 5500-5900)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = "/root/apps/TradingData"
PORTS = (5500, 5600, 5700, 5800, 5900)


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-tradingdata"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )

    def run(cmd: str) -> str:
        print("$", cmd)
        _, o, e = c.exec_command(cmd)
        out = (o.read() + e.read()).decode()
        print(out)
        return out

    run(f"cd {REMOTE} && pgrep -af 'python.*RefreshNews.py' || echo NOT_RUNNING")
    run(
        f"cd {REMOTE} && source ~/.zshrc 2>/dev/null || source ~/.bashrc 2>/dev/null; "
        f"nohup python3 RefreshNews.py >> services.log 2>&1 & sleep 3; "
        "ps aux | grep 'python3 RefreshNews.py' | grep -v grep || echo START_FAILED"
    )
    run(f"ss -lntp | grep -E ':({'|'.join(map(str, PORTS))})' || echo HTTP_PORTS_DOWN")
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
