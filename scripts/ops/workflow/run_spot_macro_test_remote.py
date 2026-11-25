#!/usr/bin/env python3
"""SSH geegoo-agent: spot-test diagnose_macro for cn_a / us / crypto (one symbol each)."""
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
LOG = "/tmp/macro_spot_test_2026-11-25.out"

RUNS = (
    ("cn_a", "000001.SH"),
    ("us", "SPY.US"),
    ("crypto", "BTC-USD.CRYPTO"),
)


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
        timeout=60,
    )
    lines = [
        f"cd {INSTALL}",
        f"set -a && . {ENV} && set +a",
        "export PYTHONUNBUFFERED=1",
        "export GEEGOO_AGENT_OP_HTTP_URL=http://127.0.0.1:3400",
        "export MACRO_API_BASE=http://146.56.225.252:3220",
        f"exec > {LOG} 2>&1",
        'echo "start $(date -u +%Y-%m-%dT%H:%M:%SZ)"',
    ]
    for wf, sym in RUNS:
        lines.append(f'echo "==== workflow={wf} symbol={sym} ===="')
        lines.append(
            f"/usr/bin/python3 -u scripts/ops/workflow/run_macro_monitor_lab_batch.py "
            f"--workflow {wf} --date auto --symbols {sym}"
        )
    lines.append('echo "done $(date -u +%Y-%m-%dT%H:%M:%SZ)"')
    remote = "bash -lc " + json.dumps(" && ".join(lines))
    bg = f"nohup {remote} </dev/null & echo PID=$! LOG={LOG}"
    print(bg)
    _, stdout, stderr = client.exec_command(bg, timeout=30)
    print(stdout.read().decode("utf-8", errors="replace"))
    err = stderr.read().decode("utf-8", errors="replace")
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())