#!/usr/bin/env python3
"""SSH：按池重跑 diagnose_macro_pool（--pool-jev-only）。"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
INSTALL = "/home/ubuntu/.geegoo/geegoo-agent"
ENV = "/home/ubuntu/.geegoo/agent.env"


def _run(client: paramiko.SSHClient, wf: str, date: str, timeout: int) -> int:
    cmd = (
        f"cd {INSTALL} && set -a && . {ENV} && set +a && "
        "export PYTHONUNBUFFERED=1 "
        "GEEGOO_AGENT_OP_HTTP_URL=http://127.0.0.1:3400 "
        "MACRO_API_BASE=http://146.56.225.252:3220 && "
        f"/usr/bin/python3 -u scripts/ops/workflow/run_macro_monitor_lab_batch.py "
        f"--workflow {wf} --pool-jev-only --date {date}"
    )
    print(f"\n=== pool-jev-only {wf} date={date} ===", flush=True)
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    if out.strip():
        print(out)
    if err.strip():
        print(err, file=sys.stderr)
    print(f"exit={code}", flush=True)
    return code


def main() -> int:
    date = sys.argv[1] if len(sys.argv) > 1 else "auto"
    workflows = sys.argv[2:] if len(sys.argv) > 2 else ["cn_a", "hk"]
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
        timeout=60,
    )
    timeout = int(os.environ.get("MACRO_POOL_JEV_SSH_TIMEOUT", str(30 * 60)))
    rc = 0
    for wf in workflows:
        rc = max(rc, _run(client, wf, date, timeout))
    client.close()
    return rc


if __name__ == "__main__":
    raise SystemExit(main())