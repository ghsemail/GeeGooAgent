#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(c: paramiko.SSHClient, cmd: str) -> str:
    _, o, e = c.exec_command(cmd, timeout=60)
    return (o.read() + e.read()).decode("utf-8", errors="replace")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

    print("=== agent-runtime direct ===")
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        for path in ["/v1/sessions?limit=3", "/v1/metrics/overview"]:
            print(run(c, f"curl -s -w '\\nHTTP:%{{http_code}}\\n' http://127.0.0.1:3400{path}"))
    finally:
        c.close()

    print("=== BFF with API key only ===")
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        print(
            run(
                c,
                "bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && "
                "curl -s -w \"\\nHTTP:%{http_code}\\n\" -H \"Authorization: Bearer $GEEGOO_BOT_AGENT_API_KEY\" "
                "http://127.0.0.1:3110/op_agent/v1/sessions?limit=3'",
            )
        )
        print(run(c, "journalctl -u geegoo-bot --no-pager -n 30 2>/dev/null | tail -20 || true"))
    finally:
        c.close()


if __name__ == "__main__":
    main()
