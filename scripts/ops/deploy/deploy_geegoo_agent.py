#!/usr/bin/env python3
"""Deploy GeeGooAgent to geegoo-agent host after git push."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
INSTALL_DIR = "/home/ubuntu/.geegoo/geegoo-agent"
BIN_DIR = "/home/ubuntu/.geegoo/bin"


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[int, str, str]:
    print(f"\n$ {cmd}", flush=True)
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print(err.rstrip(), file=sys.stderr)
    return code, out, err


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )

    steps = [
        f"cd {INSTALL_DIR} && git fetch origin main",
        f"cd {INSTALL_DIR} && git checkout main",
        f"cd {INSTALL_DIR} && git reset --hard origin/main",
        f"cd {INSTALL_DIR} && git log -1 --oneline",
        f"cd {INSTALL_DIR} && go build -o {BIN_DIR}/agentRuntimeServer ./cmd/agent-runtime",
        f"cd {INSTALL_DIR} && go build -o {INSTALL_DIR}/geegoo ./cmd/geegoo",
        f"ln -sf {INSTALL_DIR}/geegoo {BIN_DIR}/geegoo",
        f"cd {INSTALL_DIR} && bash start.sh restart-all",
        "curl -sf http://127.0.0.1:3400/health && echo",
        f"{BIN_DIR}/geegoo doctor || true",
    ]
    for cmd in steps:
        code, _, _ = run(client, cmd, timeout=900)
        if code != 0 and "doctor" not in cmd and "health" not in cmd:
            client.close()
            return code

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
