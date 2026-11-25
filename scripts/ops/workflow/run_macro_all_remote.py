#!/usr/bin/env python3
"""SSH geegoo-agent: run macro workflow all (foreground or nohup)."""
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
ENV = "/home/ubuntu/.geegoo/agent.env"


def main() -> int:
    date = sys.argv[1] if len(sys.argv) > 1 else "2026-11-24"
    bg = "--background" in sys.argv
    log = f"/tmp/macro_all_{date}.out"
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
    inner = " && ".join(
        [
            f"cd {INSTALL}",
            f"set -a && . {ENV} && set +a",
            "export PYTHONUNBUFFERED=1",
            "export GEEGOO_AGENT_OP_HTTP_URL=http://127.0.0.1:3400",
            "export MACRO_API_BASE=http://146.56.225.252:3220",
            f"exec >> {log} 2>&1",
            'echo "start $(date -u +%Y-%m-%dT%H:%M:%SZ) date=' + date + '"',
            "/usr/bin/python3 -u scripts/ops/workflow/run_macro_monitor_lab_batch.py "
            f"--workflow all --date {date}",
            'echo "done $(date -u +%Y-%m-%dT%H:%M:%SZ) exit=$?"',
        ]
    )
    if bg:
        cmd = f"nohup bash -lc {json.dumps(inner)} </dev/null & echo PID=$! LOG={log}"
        _, stdout, stderr = client.exec_command(cmd, timeout=30)
    else:
        cmd = f"bash -lc {json.dumps(inner)}"
        timeout = 8 * 3600
        _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    if bg:
        print(out.strip())
        client.close()
        return 0
    print(out)
    if err.strip():
        print(err, file=sys.stderr)
    code = stdout.channel.recv_exit_status()
    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())