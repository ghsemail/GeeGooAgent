#!/usr/bin/env python3
"""Diagnose agent-runtime reachability from bot server."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(c: paramiko.SSHClient, cmd: str, timeout: int = 60) -> str:
    print(f"\n>>> {cmd}\n")
    _, o, e = c.exec_command(cmd, timeout=timeout)
    text = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(text[-3000:])
    return text


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    bot = cfg["targets"]["geegoo-bot"]["ssh"]

    for name, s in [("agent", agent), ("bot", bot)]:
        c = paramiko.SSHClient()
        c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
        print(f"\n==== {name} ({s['host']}) ====")
        if name == "agent":
            run(c, "ss -tlnp | grep 3400 || netstat -tlnp 2>/dev/null | grep 3400 || true")
            run(c, "sudo ufw status 2>/dev/null || true")
            run(c, "curl -sf --max-time 3 http://127.0.0.1:3400/health || echo LOCAL_FAIL")
        else:
            run(c, "curl -sf --max-time 5 http://119.45.16.112:3400/health || echo REMOTE_FAIL")
            run(c, "nc -zv -w 3 119.45.16.112 3400 2>&1 || true")
        c.close()


if __name__ == "__main__":
    main()
