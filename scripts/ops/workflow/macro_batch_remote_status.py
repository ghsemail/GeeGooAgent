#!/usr/bin/env python3
"""查看 Agent 主机上宏观日批进程是否在跑。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"


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
    cmd = "ps -eo pid,etime,cmd | grep run_macro_monitor_lab_batch | grep -v grep || true"
    _, stdout, _ = client.exec_command(cmd, timeout=30)
    out = stdout.read().decode("utf-8", errors="replace").strip()
    client.close()
    if out:
        print(out)
    else:
        print("(no macro batch process)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())