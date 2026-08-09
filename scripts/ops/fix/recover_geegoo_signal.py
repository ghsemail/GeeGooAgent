#!/usr/bin/env python3
"""Recover GeeGooSignal services after catalog-api outage."""
from __future__ import annotations

import json
import time
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(client: paramiko.SSHClient, cmd: str) -> str:
    print(f">>> {cmd}")
    _, stdout, stderr = client.exec_command(cmd, timeout=180)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print("ERR:", err.rstrip())
    return out


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-tradingsignal"]["ssh"]
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(s["host"], 22, s["user"], s["password"], timeout=20)

    run(ssh, "cd /root/apps/GeeGooSignal && bash start.sh restart")
    time.sleep(8)
    run(ssh, "cd /root/apps/GeeGooSignal && bash start.sh status")
    run(ssh, "curl -s -o /dev/null -w 'catalog_health=%{http_code}\\n' http://127.0.0.1:3210/health")
    run(
        ssh,
        "curl -sI -X OPTIONS http://127.0.0.1:3210/login "
        "-H 'Origin: http://localhost:54321' "
        "-H 'Access-Control-Request-Method: POST' | head -8",
    )
    run(
        ssh,
        "curl -s -o /dev/null -w 'op_catalog_login=%{http_code}\\n' "
        "-X POST http://127.0.0.1:8088/op_catalog/login "
        "-H 'Content-Type: application/json' "
        "-H 'Authorization: Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9' "
        "-d '{\"username\":\"test\",\"password\":\"test\"}'",
    )

    ssh.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
