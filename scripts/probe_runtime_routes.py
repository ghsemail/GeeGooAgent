#!/usr/bin/env python3
"""Probe agent-runtime Cockpit routes on agent server."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(c: paramiko.SSHClient, cmd: str, timeout: int = 60) -> None:
    print(f"\n>>> {cmd}\n")
    _, o, e = c.exec_command(cmd, timeout=timeout)
    text = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(text[-3500:])


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        run(c, "ps aux | grep -E 'agent-runtime|agentRuntime' | grep -v grep || true")
        run(c, "curl -sf http://127.0.0.1:3400/health; echo")
        run(c, "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3400/v1/tools; echo")
        run(c, "curl -s http://127.0.0.1:3400/v1/tools | head -c 300; echo")
        run(c, "systemctl --user status geegoo-agent-runtime 2>/dev/null | head -20 || true")
        run(c, "ls -la ~/.geegoo/bin/ 2>/dev/null | head -10")
    finally:
        c.close()


if __name__ == "__main__":
    main()
