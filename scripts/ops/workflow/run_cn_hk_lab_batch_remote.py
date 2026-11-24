#!/usr/bin/env python3
"""SSH geegoo-agent：重跑 A 股池 (cn_a) 与港股池 (hk) 全标的 + 池级 JEV。"""
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


def _run_remote(client: paramiko.SSHClient, workflow: str, date: str, timeout: int) -> int:
    wf = workflow
    extra = ""
    if workflow == "bond_finalize":
        wf = "bond"
        extra = "--bond-pool-finalize-only "
    cmd = (
        f"cd {INSTALL} && set -a && . {ENV} && set +a && "
        "export PYTHONUNBUFFERED=1 "
        "GEEGOO_AGENT_OP_HTTP_URL=http://127.0.0.1:3400 "
        "MACRO_API_BASE=http://146.56.225.252:3220 && "
        f"/usr/bin/python3 -u scripts/ops/workflow/run_macro_monitor_lab_batch.py "
        f"{extra}--workflow {wf} --date {date}"
    )
    print(f"\n=== workflow={workflow} date={date} ===", flush=True)
    print("$", cmd, flush=True)
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
    date = "auto"
    if len(sys.argv) > 1 and sys.argv[1].strip():
        date = sys.argv[1].strip()
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
    # 单标的最坏 ~15min×重试；整池预留 3h
    per_pool_timeout = int(os.environ.get("MACRO_CN_HK_BATCH_SSH_TIMEOUT", str(3 * 3600)))
    rc = 0
    workflows = sys.argv[2:] if len(sys.argv) > 2 else ["cn_a", "hk"]
    for wf in workflows:
        rc = max(rc, _run_remote(client, wf, date, per_pool_timeout))
    client.close()
    return rc


if __name__ == "__main__":
    raise SystemExit(main())