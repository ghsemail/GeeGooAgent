#!/usr/bin/env python3
"""SSH to geegoo-agent and run crypto Lab batch once (smoke)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
INSTALL = "/home/ubuntu/.geegoo/geegoo-agent"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = resolve_ssh_cfg(cfg["targets"]["geegoo-agent"]["ssh"], "geegoo-agent")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        key_filename=ssh_cfg.get("key_file"),
        timeout=30,
    )
    cmd = (
        f"cd {INSTALL} && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && "
        "export GEEGOO_AGENT_OP_HTTP_URL=http://127.0.0.1:3400 "
        "MACRO_API_BASE=http://146.56.225.252:3220 && "
        "/usr/bin/python3 scripts/ops/workflow/run_macro_monitor_lab_batch.py --workflow crypto"
    )
    print("$", cmd, flush=True)
    _, stdout, stderr = client.exec_command(cmd, timeout=3600)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    if out.strip():
        print(out)
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())