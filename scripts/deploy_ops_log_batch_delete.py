#!/usr/bin/env python3
"""Deploy ops-log batch delete: GeeGooSignal + trading_operation web."""
from __future__ import annotations

import json
import subprocess
import sys
import time
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
DEPLOY_WEB = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\scripts\deploy_trading_operation_web.py")
FLUTTER_WEB = Path(r"D:\Geegoo\trading_operation\build\web")


def ssh_run(host: str, user: str, password: str, cmd: str) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=30)
    _, stdout, stderr = client.exec_command(cmd, timeout=300)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    client.close()
    if out:
        print(out)
    if err:
        print("STDERR:", err)
    return out


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    print("=== deploy GeeGooSignal ===")
    ssh_run(
        sig["host"],
        sig["user"],
        sig["password"],
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    )
    time.sleep(8)

    print("=== verify deleteStrategyGenerationLogs ===")
    ssh_run(
        sig["host"],
        sig["user"],
        sig["password"],
        "curl -s -X POST http://127.0.0.1:3210/deleteStrategyGenerationLogs "
        "-H 'Content-Type: application/json' "
        "-H 'Authorization: Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9' "
        "-d '{\"run_id\":\"__nonexistent__\"}'",
    )

    print("=== flutter build web ===")
    r = subprocess.run(
        ["flutter", "build", "web", "--release"],
        cwd=r"D:\Geegoo\trading_operation",
        capture_output=True,
        text=True,
    )
    if r.stdout:
        print(r.stdout[-2000:])
    if r.returncode != 0:
        print(r.stderr[-2000:] if r.stderr else "flutter build failed")
        return r.returncode

    if not FLUTTER_WEB.is_dir():
        print("build/web missing after flutter build")
        return 1

    print("=== deploy trading_operation web ===")
    r = subprocess.run([sys.executable, str(DEPLOY_WEB)], capture_output=True, text=True)
    print(r.stdout)
    if r.stderr:
        print(r.stderr)
    return r.returncode


if __name__ == "__main__":
    sys.exit(main())
