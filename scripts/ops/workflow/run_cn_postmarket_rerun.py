#!/usr/bin/env python3
"""Deploy agent fix and rerun postmarket_stock backfill."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REPORT_DATE = "2026-08-07"


def ssh_run(target: str, cmd: str, timeout: int = 900) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> int:
    install = json.loads(DEPLOY.read_text(encoding="utf-8"))["targets"]["geegoo-agent"]["install_cmd"]
    print("=== deploy ===")
    print(ssh_run("geegoo-agent", install, timeout=900))
    print(f"\n=== run postmarket_stock {REPORT_DATE} ===")
    run_cmd = (
        "export PATH=$HOME/.geegoo/bin:$PATH; "
        f"timeout 3600 $HOME/.geegoo/bin/geegoo run "
        f"--config $HOME/.geegoo/config.json --report-date {REPORT_DATE} postmarket_stock 2>&1"
    )
    print(ssh_run("geegoo-agent", run_cmd, timeout=3620))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
