#!/usr/bin/env python3
"""Deploy GeeGooAgent to geegoo-agent host after git push."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
EXPECTED_HEAD = "6a3fd62e"


def ssh_run(host_cfg: dict, cmd: str, timeout: int = 300) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=host_cfg["host"],
        port=int(host_cfg.get("port", 22)),
        username=host_cfg["user"],
        password=host_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    if code != 0:
        raise RuntimeError(f"exit {code}: {err.strip() or out.strip()}")
    return out


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    repo = "/home/ubuntu/.geegoo/geegoo-agent"

    cmds = [
        f"cd {repo} && git fetch origin main && git reset --hard origin/main",
        f"cd {repo} && git log -1 --oneline",
        f"cd {repo} && bash start.sh build",
        f"cd {repo} && bash start.sh restart-runtime",
        f"cd {repo} && bash start.sh status",
        "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1",
        "curl -sf http://127.0.0.1:3400/health && echo",
    ]
    for cmd in cmds:
        print(f"\n>>> {cmd}")
        print(ssh_run(agent, cmd))

    head = ssh_run(agent, f"cd {repo} && git rev-parse --short HEAD").strip()
    if not head.startswith(EXPECTED_HEAD[:7]):
        print(f"WARN: expected {EXPECTED_HEAD}, got {head}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
